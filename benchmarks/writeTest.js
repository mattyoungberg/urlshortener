import { check } from 'k6';
import { SharedArray } from 'k6/data';
import { scenario } from 'k6/execution';
import http from 'k6/http';
import { URL } from 'https://jslib.k6.io/url/1.0.0/index.js';

const data = new SharedArray('data', function () {
    return JSON.parse(open('./fake_urls.json')).urls;
});

export const options = {
    scenarios: {
        'write-test': {
            executor: 'shared-iterations',
            vus: 8,
            iterations: data.length,
        }
    }
};

export default function () {
    const longUrl = data[scenario.iterationInTest];
    const url = new URL('http://localhost:8080/api/v1/shorten');
    url.searchParams.append('longUrl', longUrl);
    const res = http.post(url.toString(), null, { tags: { name: 'api/v1/shorten' } });

    const result = check(res, {
        'HTTP Status 200': (r) => r.status === 200,
    });

    if (!result) {
        console.error(`Request failed w/ status: ${res.status}`);
    }
}
