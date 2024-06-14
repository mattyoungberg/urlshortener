# URLShortner

## Goal

In this personal project, I set out to learn about services that generate shortened
URLs by implementing a simple one myself. They do this through hash functions that have
to have some pretty fascinating characteristics. Keep reading to learn more about how I
did it.

## Performance Goals

Ultimately, I based my performance goals off of an example scenario given in _System
Design_ by Alex Xu:

- Support 100 million new URLs per day (or roughly 1,160 a second)
- Support a ratio of 10:1 reads (or redirects) for every write, so 11,600 reads a second
- Support being able to generate URLs for at least 10 years
- Assume the average URL length is 100 and keep storage under 36.5TB

After explaining how I wrote the service and configured the database, I'll review how I
did on each of these goals in the [How Did I Do](#how-did-i-do) section.

## Getting the shortened url

To get a shortened URL, you need some kind of identifier that you can put through a hash
function. You'd think that you'd simply hash the URL, but the two [sources](#sources)
that I used both suggested that instead of hashing the URL directly, you instead created
a unique identifier to attach to every URL that came in to be shortened. It was then
this ID instead that you would hash.

For the sake of this project, there were two routes that I could take:

1. **Random value hashing**: This idea is simple: generate a random value, hash it to
some fixed length, make sure there are no collisions, and then persist out the mapping
of the original url, the hashed version, and the ID that you ultimately put through your
hash function. Pros? It's simple and easy to implement. Cons? It's not very performant,
as you have to go to the DB to check for collisions and regenerate the hash if there is
one.

2. **Unique ID generation**: This idea is harder to implement, but you can beat the 
performance of random value hashing if you can assure that the IDs you generate are 
unique every time (due to not having to check for collisions).
[Twitter's Snowflake](https://tinyurl.com/3n25yhat) is a great example of doing this in 
a distributed environment. In Snowflake, each unique ID is a 64-bit number that encodes
various pieces of information, including a milliseconds since a custom epoch, signatures
of the database and service that generated the ID, and more. This approach guarantees
uniqueness and even sortability, but one of the downsides is, aside from complexity, is
that you also run into the situation that IDs can potentially be guessed (especially if
you're only running a single instance of both the app and the database, like I am).

Ultimately, I decided to go with the second approach due to its performance benefits.

## Base62 Encoding, Hash Length, and ID Size

After you've generated a unique ID, you need to hash it. I decided to hash the unique
IDs using a base62 encoding scheme, which is comparable to base64 encoding, but with
non-alphanumeric characters to be removed (the sum of all characters that match
\[0-9]\[a-z]\[A-Z] is 62). The main motivation for this decision is asethetic, and the
fact that when you double-click the hash in your browser, it will select the entire
string, not truncating the highlight upon encountering a non-alphanumeric character.
It would bother me to no end if I couldn't highlight the entire hash and no more than
the hash with a quick double-click.

Initially, I was curious if I could use a base62 library for encoding my IDs. I tried
two: one was based on big ints, and the other was based on variadic length encoding. The
short of it was that both these libraries were too general-purpose, and ultimately left
me with more downsides that percolated through the design of my IDs and in assuring the
performance of my service. So I decided to write my own base62 encoding algorithm.

Conceptually, it's simple. You take a bit-word and do decimal division on it. When you
divide by 62, you're left with a quotient and a remainder. These two numbers are the
next digits in your base62 encoded string. You continue this process until you've
processed the entire string of bits.

In order to make this approach work, there is an upper limit on the size of the word
that you can use. If the word was fully on, you could have a maximum quotient of 61 and
a remainder of 61. This means the maximum size of the word that you can encode is 62^2 -
1, or 3843. This is a 12-bit number when converted to binary (1111 0000 0011). To avoid
overflow, we floor the word size to 11 bits, the max being 2047 in decimal.

Next, we need to decide how many bits we want to encode, which, ideally, would be a
multiple of our 11 bit word derived from the last step. I decided to go with 55 bits
because, based on the encoding scheme derived below, would last almost a century and
permit writing 8,388,608 unique IDs per second. Let me explain.

## Unique ID Encoding Scheme

To encode a 55 bit ID, we'll use the number of seconds since the UNIX epoch and a 
sequence number. However, we'll pad it with a zero in the middle to make the encoding of
the ID a bit easier (it can act as the most significant bit of the sequence number to
make it a power of eight without affecting its value).

At the time of writing, the seconds since the epoch was 1,714,085,905. This can be fit
in a 32-bit unsigned integer no problem, and will allow this app to continue working
until [February 7th, 2106](https://tinyurl.com/pz9wukun).

(Note: if I wanted to be able to get a half century more out of this service, I could
use a custom epoch based on when the service is published, but I'll probably be dead by
2106, so it's not my problem, sorry.)

With the remaining 23 bits, we create a sequence number that will increment every time
we generate a new ID. This means that the service can generate 8,388,608 new ids per
second, or 724,775,731,200 a day.

The encoding scheme can be visualized like this:

```plaintext
+-----------------------------------------+---------+------------------------------+
|              Unix Timestamp             | Padding |           Sequence           |      
|               (32 bits)                 | (1 bit) |           (23 bits)          |     
+-----------------------------------------+---------+------------------------------+
| 0000 0000 0000 0000 0000 0000 0000 0000 |    0    | 000 0000 0000 0000 0000 0000 | 
+-----------------------------------------+---------+------------------------------+
```

## Concurrency

Because the sequence has to reset once every second, we need some kind of timer that can
intercede in the `UniqueIDGenerator` implementation and reset it once a second. The
idiom in Golang is to use a `Timer` type which writes to a channel once every second. We
kick this out to another goroutine, but there is a critical section of the code where we
don't want this "sequence resetting" thread and the main "id generation" thread
overlapping: whenever the sequence number is actively being used.

In order to control access to this critical section, we use a simple `Lock`. Whenever
we're asked to grab an ID, we acquire the lock. But before we go on to generate it, we
make sure that the sequence isn't exhausted; if it is, we sleep the thread and
relinquish the lock until we're reawoken to check the condition. Once we're good to move
on, we create the ID and increment the sequence before reliquishing the lock. When the
sequence-resetting thread gets a tick from the `Timer` via its channel, it acquires the
lock guarding the current sequence number and updates it back to zero.

Only once the id-generating thread is woken _and_ finds that the sequence number is
exhausted does it go ahead and start generating the values for the unique ID,
guaranteeing that, for example, it's not pulling the seconds since the epoch _before_
generating the sequence number, in which case, we could accidentally create a duplicate.
In that sense, the whole id-generating function is a critical section.

So, in all, there are two uses of concurrency here:

1) Creating another process thread (goroutine) in order to manage the ticks from a
`Timer`, and
2) Using a `Lock` synchronization primitive in order to control which thread is allowed
to write to the sequence at any given point.

Through it's use, we assure we never generate a duplicate ID, and when faced with
volumes we can't handle, we put off creating the unique ID until we can do so safely.

## Database

For the sake of benchmarking, I'm deploying this alongside an instance of Percona Server, which is a fork of MySQL.
Admittedly, I probably would've gone with a vanilla instance of MySQL if they had exposed the ability to change the
underlying storage engine from InnoDB to RocksDB as easily as Percona did (which simply required setting an environment
variable). RocksDB is a key-value store that is optimized for write-heavy workloads on SSDs, and as I profiled both the
app and  the queries the app issued against the database, those writes ended up taking the bulk of the response time.
With an index on the `longUrl` column of the table, the read speed using RocksDB wasn't notably slower than InnoDB, but
write speed notably improved.

In the end, my choice of DB was based on hitting the read/write speeds outlined in the 
[Performance Goals](#performance-goals) section.

## Testing

I'm running these benchmarks on my own personal machine:

- **OS**: Ubuntu 22.04 LTS
- **Processor**: 12th Gen Intel Core i7-12700K (12 cores, 20 threads). No overclocking enabled.
- **Memory**: 64GB DDR5 @ 5200MHz
- **Storage**: 500GB NVMe SSD, 1900MB/s writes, 2400MB/s reads

However, it's worth noting that I'm sandboxing the Docker containers. The app only gets access to 2 cores and 4GB of
memory, while the DB gets 6 cores and 32GB of memory.

Docker compose was used for deployment.

K6 was used for load testing. You can see the [writeTest.js](./writeTest.js) and [readTest.js](./readTest.js) in the
root of the project to see how those were ran. The read test would happen after seeding the app via
[seed.py](./seed.py).

## How Did I Do?

### Write speed: 1,160 URLs per second

For a single-instance, single-db app, I was still able to achieve average write speeds of ~970 URLs per second for the
first 100,000 urls. This is fairly close to the goal of 1,160 URLs per second. I'm not quite sure what the internal
storage structure is for RocksDB, but speed seemed to correspond to some kind of logarithmic growth, so I'd assume that
it'd grow similar to InnoDB, which uses B+ trees to structure tables.

Since the benchmarks were set forth assuming a distributed app and a distributed database, I'll call it an unfair fight
and conclude saying I achieved fairly close results here.

### Read speed: 11,600 URLs per second

The app+db was able to knock this out of the water: my read test is able to handle ~15,500 reads per second leveraging
an index on the `longUrl` column. This is quite bit more than the 11,600 URLs per second that I was aiming for, even
with a storage engine optimized for writes.

### 10 years of URL generation

As covered in [Unique ID Encoding Scheme](#unique-id-encoding-scheme), the service can generate unique IDs for almost
a century. This is well beyond the 10 years that I was aiming for.


### Storage under 36.5TB

I admittedly don't know how to measure this that well, but I can do some "back of the napkin" calculations:

A single record in my `urls` table takes two values: 1) a seven-byte unique identifier, and 2) a URL that, on average,
will be 100 characters in length.

Assuming that the URL adheres to ASCII encoding, the size of a given record would be 107 bytes (without accounting for
indexes). If we're generating 100 million URLs a day, that's 10,700,000,000 bytes, or 10.7GB a day. Over the course of a
year, that's 3.9TB. Over the course of 10 years, that's 39TB.

That's about 3TB over the ask, but I also understand that underlying storage engines potentially compress data, and
RocksDB also advertises its superior ability to compress data compared to InnoDB.

(I did some cursory queries both when the table was empty and when it was seeded w/ 100,000 records. Both times it
claimed its size was ~23MB of data... So color me confused.)

I would assume that, with compression, the storage of the table would require slightly less than 36.5TB, and so I'll
call that a (nuanced) win. If anyone knows how to estimate this better, please reach out to me.

I'm not exactly sure where the _System Design_ book got its storage benchmark from, since it was designing for an 8 byte
identifier and a 100 byte URL. It's somewhat implied that these scheme would keep it under 36.5TB, but since I designed
an ID that uses one less byte, the math doesn't quite add up.

## Limitations

- **The encoding scheme is predictable.** Anyone with basic knowledge of baseX encoding
can see how the IDs are incrementing and could guess at how it's doing it. If this were 
being designed for an actual production environment, a more secure encoding scheme would
have to be derived.
- **Only a single instance of this network service can run.** If more than one instance
were deployed, we'd probably want to reengineer the ID scheme to add 8 more bits and
include signatures of the instance that generated the ID to maintain uniqueness. If the
DB were to be distributed as well, we'd want a signature from the DB too when encoding
the ID.

## Sources

- [The System Design Interview](https://tinyurl.com/4ktudfyd) by Alex Xu
- [Coding Challenges: Build Your Own URL Shortener](https://tinyurl.com/3mh6k2xw) by
John Crickett