function Ocean(api) {
  this.api = api;
}

Ocean.prototype = {
  orders: function (callback, market, offset) {
    this.api.request('GET', 'https://events.ocean.one/orders?state=PENDING&order=DESC&limit=100&market=' + market + '&offset=' + offset, undefined, function (resp) {
      return callback(resp);
    });
  },

  ticker: function (callback, market) {
    this.api.request('GET', 'https://events.ocean.one/markets/' + market + '/ticker', undefined, function (resp) {
      return callback(resp);
    });
  },

  trades: function (callback, market, offset) {
    this.api.request('GET', 'https://events.ocean.one/markets/' + market + '/trades?order=DESC&limit=100&offset=' + offset, undefined, function (resp) {
      return callback(resp);
    });
  }
};

export default Ocean;
