import json

import requests


def main():
    s = requests.Session()
    res = s.get("http://localhost:8080/api/v1/health")
    if res.status_code != 200:
        print("Service is not up")
        return

    with open("./fake_urls.json") as f:
        urls = json.load(f)["urls"]

    for url in urls:
        res = s.post("http://localhost:8080/api/v1/shorten", params={"longUrl": url})
        if res.status_code != 200:
            print(f"Failed to shorten {url}")
            continue


if __name__ == "__main__":
    main()
