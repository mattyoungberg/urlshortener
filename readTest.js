import { check } from 'k6';
import { SharedArray } from 'k6/data';
import { scenario } from 'k6/execution';
import http from 'k6/http';
import { URL } from 'https://jslib.k6.io/url/1.0.0/index.js';

const data = new SharedArray('data', function () {
    return JSON.parse(open('./shortened_urls.json'));
});

export const options = {
    scenarios: {
        'read-test': {
            executor: 'shared-iterations',
            vus: 8,
            iterations: data.length,
        }
    }
};

export default function () {
    const combo = data[scenario.iterationInTest];
    const shortUrl = combo["shortUrl"]
    const longUrlExpected = combo["longUrl"]

    const url = new URL('http://localhost:8080/api/v1/shortUrl');
    url.searchParams.append('shortUrl', shortUrl);
    const res = http.get(url.toString(), { tags: { name: 'api/v1/shortUrl' } });

    const result = check(res, {
        'HTTP Status 200': (r) => r.status === 200,
        'Redirected to correct URL': (r) => JSON.parse(r.body).longUrl === longUrlExpected,
    });

    if (!result) {
        console.error(`Request failed w/ status: ${res.status} or incorrect redirect: ${res.body}`);
    }
}
