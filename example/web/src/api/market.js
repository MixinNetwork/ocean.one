function Market(api) {
  this.api = api;
}

Market.prototype = {
  index: function (callback) {
    this.api.request('GET', '/markets', undefined, function (resp) {
      return callback(resp);
    });
  },

  market: function (callback, market) {
    this.api.request('GET', '/markets/' + market, undefined, function (resp) {
      return callback(resp);
    });
  },

  like: function (callback, market) {
    this.api.request('POST', '/markets/' + market + '/like', undefined, function (resp) {
      return callback(resp);
    });
  },

  dislike: function (callback, market) {
    this.api.request('POST', '/markets/' + market + '/dislike', undefined, function (resp) {
      return callback(resp);
    });
  },

  candles: function (callback, market, granularity) {
    this.api.request('GET', '/markets/' + market + '/candles/' + granularity, undefined, function (resp) {
      return callback(resp);
    });
  }
};

export default Market;
