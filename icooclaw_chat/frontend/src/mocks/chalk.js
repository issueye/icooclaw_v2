function createChalkFn() {
  const fn = function(...args) {
    if (args.length === 0) return '';
    if (typeof args[0] === 'string') return args[0];
    if (Array.isArray(args[0])) {
      return args[0].reduce((result, str, i) => {
        return result + str + (args[i + 1] !== undefined ? args[i + 1] : '');
      }, '');
    }
    return String(args[0]);
  };
  return new Proxy(fn, {
    get(target, prop) {
      if (prop === 'default') return createChalkFn();
      return createChalkFn();
    }
  });
}

const chalkMock = createChalkFn();

export default chalkMock;
export const chalk = chalkMock;
