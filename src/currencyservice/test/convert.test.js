/*
 * Regression tests for CurrencyService/Convert (HBL-103).
 */

process.env.DISABLE_PROFILER = '1';

const test = require('node:test');
const assert = require('node:assert');
const { convert } = require('../server');
const rates = require('../data/currency_conversion.json');

function convertAsync (from, toCode) {
  return new Promise((resolve, reject) => {
    convert({ request: { from, to_code: toCode } }, (err, result) => {
      if (err) return reject(err);
      resolve(result);
    });
  });
}

function usd (units, nanos) {
  return { currency_code: 'USD', units, nanos };
}

test('carries the fractional part of converted units into nanos', async () => {
  const cases = [
    [usd(19, 990000000), 'GBP', 15, 201594869],
    [usd(2, 490000000), 'GBP', 1, 893545334],
    [usd(19, 990000000), 'JPY', 2235, 60592707],
    [usd(19, 990000000), 'EUR', 17, 682441398]
  ];
  for (const [from, to, units, nanos] of cases) {
    const result = await convertAsync(from, to);
    assert.deepStrictEqual(result, { currency_code: to, units, nanos }, `${from.units}.${from.nanos} USD -> ${to}`);
  }
});

test('USD to USD returns the amount unchanged', async () => {
  const result = await convertAsync(usd(19, 990000000), 'USD');
  assert.deepStrictEqual(result, { currency_code: 'USD', units: 19, nanos: 990000000 });
});

test('matches the rate table for every currency', async () => {
  const prices = [usd(0, 990000000), usd(2, 490000000), usd(19, 990000000), usd(101, 960000000), usd(1234, 500000000)];
  for (const to of Object.keys(rates)) {
    for (const from of prices) {
      const result = await convertAsync(from, to);
      const amount = from.units + from.nanos / 1e9;
      const expected = amount / rates.USD * rates[to];
      const actual = result.units + result.nanos / 1e9;

      assert.ok(Number.isInteger(result.units), `${to}: units is an integer`);
      assert.ok(Number.isInteger(result.nanos), `${to}: nanos is an integer`);
      assert.ok(result.nanos >= 0 && result.nanos <= 999999999, `${to}: nanos in range`);
      assert.ok(Math.abs(actual - expected) < 1e-6 * Math.max(1, expected), `${amount} USD -> ${to}: got ${actual}, want ${expected}`);
    }
  }
});
