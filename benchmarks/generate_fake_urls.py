import json
import random

URL_NUMBER = 100_000
DOMAIN_LENGTH = 100 - 8  # Avg is 100, 8 is the length of "www." and ".com"
CHARSET = "abcdefghijklmnopqrstuvwxyz"


def main():
    url_set = set()
    while len(url_set) != URL_NUMBER:
        url_set.add(f"www.{random_string()}.com")
    j_dict = {"urls": list(url_set)}
    with open("fake_urls.json", "w") as f:
        json.dump(j_dict, f)


def random_string():
    chars = []
    for _ in range(DOMAIN_LENGTH):
        chars.append(CHARSET[random.randint(0, 25)])
    return "".join(chars)


if __name__ == "__main__":
    main()
