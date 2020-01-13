const {calculatePct} = require('./calc');
const {expect} = require('chai');
const { sprintf } = require('sprintf-js');

describe('calculatePct', () => {
    it('should return a positive pct if current more than previous', () => {
        const actual = calculatePct(120, 100);
        expect(actual).to.equal(20);

    });
    it('should return a negative pct if current more than previous', () => {
        const actual = calculatePct(80, 100);
        expect(actual).to.equal(-20);
    });

    it('should return zero pct if current equals previous', () => {
        const actual = calculatePct(120, 120);
        expect(actual).to.equal(0);

    });
    it('should return zero pct if previous is zero or missing', () => {
        let actual = calculatePct(120, 0);
        expect(actual).to.equal(0);
        actual = calculatePct(120);
        expect(actual).to.equal(0);

        console.log(sprintf("%+.1f%%", calculatePct(100,120)));
        console.log(sprintf("%+.1f", calculatePct(120,120)));
        console.log(sprintf("%+.1f", calculatePct(120,100)));

    });
});