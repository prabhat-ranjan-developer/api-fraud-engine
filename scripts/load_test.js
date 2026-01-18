import http from 'k6/http';
import { check, sleep } from 'k6';

// --- CONFIGURATION ---
export const options = {
    insecureSkipTLSVerify: true, // Fixes proxy issues
    scenarios: {
        // Scenario 1: Normal user traffic (Steady flow)
        legitimate_traffic: {
            executor: 'constant-vus',
            vus: 20,
            duration: '30s',
            exec: 'legitimateUser', // Function to run
        },
        // Scenario 2: Attackers trying to brute force (High RPS per user)
        velocity_attack: {
            executor: 'ramping-vus',
            startVUs: 0,
            stages: [
                { duration: '10s', target: 10 }, // Ramp up to 10 attackers
                { duration: '10s', target: 0 },  // Ramp down
            ],
            exec: 'attackerUser',
        },
        // Scenario 3: Testing the new V2 API
        v2_analysis: {
            executor: 'constant-vus',
            vus: 10,
            duration: '30s',
            exec: 'v2User',
        },
    },
};

const BASE_URL = 'http://fraud-engine-app:8080';
const HEADERS = { 'Content-Type': 'application/json' };

// --- HELPER ---
function randomString(length) {
    const charset = 'abcdefghijklmnopqrstuvwxyz0123456789';
    let res = '';
    for (let i = 0; i < length; i++) {
        res += charset[Math.floor(Math.random() * charset.length)];
    }
    return res;
}

// --- SCENARIO 1: LEGITIMATE USER (V1) ---
export function legitimateUser() {
    const userId = `good_user_${Math.floor(Math.random() * 10000)}`;
    const payload = JSON.stringify({
        transaction_id: `tx_${randomString(10)}`,
        user_id: userId,
        amount: Math.floor(Math.random() * 100),
        ip_address: "192.168.1.50"
    });

    const res = http.post(`${BASE_URL}/api/v1/fraud/check`, payload, { headers: HEADERS });

    check(res, {
        'V1 Legitimate: status is 200': (r) => r.status === 200,
        'V1 Legitimate: latency < 100ms': (r) => r.timings.duration < 100,
    });

    sleep(1); // Normal users wait between actions
}

// --- SCENARIO 2: ATTACKER (V1) ---
export function attackerUser() {
    // Reuse the same User ID to trigger velocity rules
    const userId = `attacker_${__VU}`; // __VU is the Virtual User ID
    const payload = JSON.stringify({
        transaction_id: `tx_attack_${randomString(8)}`,
        user_id: userId,
        amount: 5000,
        ip_address: "10.0.0.66"
    });

    const res = http.post(`${BASE_URL}/api/v1/fraud/check`, payload, { headers: HEADERS });

    // We expect EITHER 200 (first 5 requests) OR 403 (blocked)
    check(res, {
        'V1 Attack: status 200 or 403': (r) => r.status === 200 || r.status === 403,
    });

    sleep(0.1); // Attackers spam requests quickly
}

// --- SCENARIO 3: V2 USER (V2) ---
export function v2User() {
    const userId = `v2_user_${Math.floor(Math.random() * 500)}`;
    const payload = JSON.stringify({
        transaction_id: `tx_v2_${randomString(10)}`,
        user_id: userId,
        amount: 250,
        ip_address: "172.16.0.1"
    });

    const res = http.post(`${BASE_URL}/api/v2/fraud/check`, payload, { headers: HEADERS });

    check(res, {
        'V2 Analysis: status is 202': (r) => r.status === 202, // Expect ACCEPTED
    });

    sleep(1);
}