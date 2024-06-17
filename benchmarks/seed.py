import threading
import json
import queue
import requests


class Worker(threading.Thread):

    def __init__(self, q: queue.Queue, results: list):
        super().__init__()
        self.q = q
        self.s = requests.Session()
        self.results = results

    def run(self):
        while not self.q.empty():
            url = self.q.get()
            res = self.s.post("http://localhost:8080/api/v1/shorten", params={"longUrl": url})
            res.raise_for_status()
            short_url = res.json()["shortUrl"]
            self.results.append({"shortUrl": short_url, "longUrl": url})
            self.q.task_done()


def main():
    q = queue.Queue()
    results = []
    with open("./fake_urls.json") as f:
        urls = json.load(f)
        for url in urls["urls"]:
            q.put(url)

    workers = []
    for _ in range(8):
        worker = Worker(q, results)
        worker.start()
        workers.append(worker)

    for worker in workers:
        worker.join()

    with open("./shortened_urls.json", "w") as f:
        json.dump(results, f, indent=2)


if __name__ == "__main__":
    main()
