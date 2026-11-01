const http = require('http');

function postJson(path, data, headers = {}) {
  return new Promise((resolve, reject) => {
    const payload = JSON.stringify(data);
    const req = http.request(
      {
        hostname: 'localhost',
        port: 9770,
        path,
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Content-Length': Buffer.byteLength(payload),
          ...headers,
        },
      },
      (res) => {
        let body = '';
        res.on('data', (chunk) => (body += chunk));
        res.on('end', () => {
          try {
            resolve({ status: res.statusCode, data: JSON.parse(body) });
          } catch (e) {
            resolve({ status: res.statusCode, body });
          }
        });
      }
    );
    req.on('error', reject);
    req.write(payload);
    req.end();
  });
}

async function seed() {
  console.log('--- Publishing Realistic Events ---');

  const events = [
    {
      event_type: 'payment.succeeded',
      payload: {
        payment_id: 'pay_9f82x1a9bc4',
        amount: 29900,
        currency: 'usd',
        customer: {
          id: 'cus_m847x92',
          email: 'sarah.connor@cyberdyne.io',
          name: 'Sarah Connor',
        },
        payment_method: 'card_visa_4242',
        receipt_url: 'https://pay.acme.io/receipts/rcpt_9f82x1a9bc4',
      },
    },
    {
      event_type: 'invoice.created',
      payload: {
        invoice_id: 'inv_88294a02',
        customer_id: 'cus_m847x92',
        subtotal: 29900,
        tax: 2691,
        total: 32591,
        due_date: '2026-11-15T00:00:00Z',
        lines: [
          { description: 'Enterprise Tier Platform License (1 yr)', amount: 29900 },
        ],
      },
    },
    {
      event_type: 'customer.subscription.updated',
      payload: {
        subscription_id: 'sub_88329x',
        plan: 'enterprise_scale',
        status: 'active',
        seats: 50,
        billing_cycle: 'annual',
      },
    },
    {
      event_type: 'deployment.promoted',
      payload: {
        service: 'checkout-api',
        version: 'v2.18.4',
        environment: 'production',
        sha: 'bd52293',
        health_checks: 'passing',
      },
    },
    {
      event_type: 'order.created',
      payload: {
        order_id: 'ord_91823746',
        items: [
          { sku: 'HW-RELAY-EDGE', qty: 2, price: 14900 },
          { sku: 'CBL-OPTIC-10M', qty: 4, price: 2500 },
        ],
        shipping_method: 'overnight',
        status: 'processing',
      },
    },
    {
      event_type: 'fulfillment.dispatched',
      payload: {
        fulfillment_id: 'ful_472819',
        tracking_code: 'FX-84920192',
        carrier: 'FedEx Express',
        warehouse: 'US-EAST-ORD',
      },
    },
    {
      event_type: 'payment.refunded',
      payload: {
        refund_id: 'rf_1294821',
        payment_id: 'pay_9f82x1a9bc4',
        amount: 5000,
        reason: 'customer_request',
      },
    },
    {
      event_type: 'customer.created',
      payload: {
        customer_id: 'cus_894819a',
        email: 'alex@mtlglabs.space',
        tier: 'developer',
      },
    },
  ];

  for (let i = 0; i < events.length; i++) {
    const ev = events[i];
    const res = await postJson('/api/v1/publish', ev, {
      'Idempotency-Key': `idem_${Date.now()}_${i}`,
    });
    console.log(`Published [${res.status}] ${ev.event_type} -> message_id: ${res.data?.message_id}, deliveries_queued: ${res.data?.deliveries_queued}`);
  }

  console.log('✓ Seeding events finished. Waiting for worker processing...');
  await new Promise((r) => setTimeout(r, 2000));
}

seed().catch(console.error);
