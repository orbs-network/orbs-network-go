package javascript

const SDK_JS_IMPLEMENTATION = `
$sdk = {
	state: class {
		static readUint64ByKey(key) {
			return 17;
		}
		static writeUint64ByKey(key, value) {
			return;
		}
	},
};
`
