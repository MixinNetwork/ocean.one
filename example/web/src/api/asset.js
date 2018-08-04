function Asset(api) {
  this.api = api;
}

Asset.prototype = {
  preset: function () {
    return require('./assets.json');
  },

  get: function (sym) {
    var assets = this.preset();
    for (var i = 0; i < assets.length; i++) {
      if (assets[i].symbol === sym) {
        return assets[i];
      }
    }
    return undefined;
  },

  market: function (sym) {
    var ss = sym.split('-');
    if (ss.length !== 2) {
      return undefined;
    }
    var b = ss[0], q = ss[1];
    if (b === q) {
      return undefined;
    }
    switch (q) {
      case 'USDT':
        break;
      case 'BTC':
        if (b === 'USDT') {
          return undefined;
        }
        break;
      case 'XIN':
        if (b === 'USDT' || b === 'BTC') {
          return undefined;
        }
        break;
      default:
        return undefined;
    }
    var base = this.get(b);
    var quote = this.get(q);
    if (base && quote) {
      return [base, quote];
    }
    return undefined;
  }
};

export default Asset;
