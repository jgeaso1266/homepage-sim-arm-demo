// motion-tools/lib references Worker at module load (its PCD worker). Node has
// no Worker, so stub a no-op for unit tests that only touch the proto helpers.
if (typeof (globalThis as { Worker?: unknown }).Worker === 'undefined') {
	;(globalThis as { Worker: unknown }).Worker = class {
		postMessage() {}
		terminate() {}
		addEventListener() {}
		removeEventListener() {}
	}
}
