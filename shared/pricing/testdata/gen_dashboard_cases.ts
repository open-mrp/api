import { QuantityUtils } from '../../../../dashboard/packages/objects/src/classes/measures/Quantity.ts';
import { Decimal, toDecimal } from '../../../../dashboard/packages/objects/src/classes/measures/Decimal.ts';

// Wide enough to stand in for exact arithmetic, to find where the dashboard's 40 digits change the cent.
const Exact = Decimal.clone({ precision: 200 });

// Generates dashboard_cases.json: amounts and cents computed by the dashboard's own
// QuantityUtils.multiplyRate, which pricing.ExtendedPrice ports. Regenerate from api/ with
//
//   bun shared/pricing/testdata/gen_dashboard_cases.ts > shared/pricing/testdata/dashboard_cases.json
//
// Deterministic PRNG so the cases are the same every run.
let seed = 42;
const rand = () => ((seed = (seed * 1103515245 + 12345) % 2147483648) / 2147483648);
const pick = <T,>(xs: T[]) => xs[Math.floor(rand() * xs.length)];

const ratios: [number, number][] = [[1, 1], [2, 1], [24, 1], [12, 1], [20, 1], [10, 1], [8, 1], [40, 1], [1, 7], [3, 7], [1, 3], [5, 12], [0.45359237, 1], [453.59237, 1], [1, 1000], [7, 3]];
const qtys = ['1', '3', '6', '7', '11', '1200', '0.5', '2.25', '1199.5', '0.333', '13', '100000', '17.123456789'];
const prices = ['22.5', '0.01', '0.005', '29.95', '19.95', '8.5', '1', '0.07', '3.3333', '-10', '1234.5678', '0.0125', '270', '0', '4.25', '9.99'];

const unit = (id: string, [rn, rd]: [number, number]) => ({
  id, abbreviation: id, name: id, type: 'quantity' as any,
  ratioNumerator: rn, ratioDenominator: rd, offsetNumerator: 0, offsetDenominator: 1,
  isBaseUnit: rn === 1 && rd === 1,
});
const dollar = { id: 'usd', abbreviation: '$', name: 'Dollar', type: 'currency' as any, ratioNumerator: 1, ratioDenominator: 1, offsetNumerator: 0, offsetDenominator: 1, isBaseUnit: true };

const out: any[] = [];
for (let i = 0; i < 20000; i++) {
  const qr = pick(ratios), pr = pick(ratios);
  const same = rand() < 0.2;
  const qty = pick(qtys), price = pick(prices);
  const qu = unit('q', qr);
  const du = same ? qu : unit('d', pr);
  const quantity = { id: 'x', measure: toDecimal(qty), unit: qu } as any;
  const rate = { id: 'r', measure: toDecimal(price), numeratorUnit: dollar, denominatorUnit: du } as any;
  const res = QuantityUtils.multiplyRate(quantity, rate);
  out.push({ same, qty, price, qr: qr.map(String), pr: (same ? qr : pr).map(String), amount: res.measure.toString(), cents: res.measure.toDP(2).toFixed(2) });
}
// Every case where the dashboard's 40 digits round to a different cent than exact arithmetic would,
// then samples of the other non-terminating cases and of the terminating ones.
const exactCents = (c: any) => {
  const [qn, qd, pn, pd] = [...c.qr, ...c.pr];
  return new Exact(c.qty).times(c.price).times(qn).times(pd).div(new Exact(qd).times(pn)).toDP(2).toFixed(2);
};
const isLong = (c: any) => c.amount.replace(/[-.]/g, '').length > 20;
const disagree = out.filter(c => exactCents(c) !== c.cents);
const long = out.filter(c => isLong(c) && exactCents(c) === c.cents).slice(0, 300);
const short = out.filter(c => !isLong(c)).slice(0, 200);
console.log('[\n' + [...disagree, ...long, ...short].map(c => JSON.stringify(c)).join(',\n') + '\n]');
