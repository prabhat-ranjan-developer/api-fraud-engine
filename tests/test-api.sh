#!/bin/bash

API_URL="http://4.157.220.83/api/v1/fraud/check"

echo "🚀 Starting Traffic Generator for Grafana Dashboard..."

# 1. Generate Legitimate Traffic
echo "✅ Sending Legitimate Traffic..."
for i in {1..30}; do
  curl -s -X POST $API_URL \
    -H "Content-Type: application/json" \
    -d "{\"transaction_id\": \"txn_good_$i\", \"user_id\": \"good_user_$i\", \"amount\": 150.00, \"ip_address\": \"192.168.1.10\"}" > /dev/null
  sleep 0.2
done

# 2. Trigger Velocity Limit Alert
# Sending 6 requests for the SAME user quickly to breach the 5-request limit
echo "⚠️ Triggering Velocity Limit (Spam Attack)..."
for i in {1..6}; do
  curl -s -X POST $API_URL \
    -H "Content-Type: application/json" \
    -d "{\"transaction_id\": \"txn_spam_$i\", \"user_id\": \"spammer_user_99\", \"amount\": 5000.00, \"ip_address\": \"10.0.0.5\"}" > /dev/null
done

echo "⚠️ Triggering Velocity Limit (Spam Attack)..."
for i in {1..6}; do
  curl -s -X POST $API_URL \
    -H "Content-Type: application/json" \
    -d "{\"transaction_id\": \"txn_spam_$i\", \"user_id\": \"spammer_user_101\", \"amount\": 7000.00, \"ip_address\": \"11.0.0.5\"}" > /dev/null
done


# 3. Trigger Blacklist Alert (Spikes the Redis Stream panel)

echo "🛑 Triggering Blacklist Block..."
curl -s -X POST $API_URL \
    -H "Content-Type: application/json" \
    -d '{"transaction_id": "txn_hack_1", "user_id": "bad_hacker_1", "amount": 9999.00, "ip_address": "1.1.1.1"}' > /dev/null

# 4. Generate API Errors
# Sending bad JSON (missing a quote) to trigger a 400 Bad Request
echo "❌ Generating API Errors (Bad JSON)..."
for i in {1..5}; do
  curl -s -X POST $API_URL \
    -H "Content-Type: application/json" \
    -d '{"transaction_id": "txn_bad", "user_id": ' > /dev/null
done

echo "🎉 Done! Check your Grafana dashboard."