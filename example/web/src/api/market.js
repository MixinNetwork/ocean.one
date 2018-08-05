function Market(api) {
  this.api = api;
}

Market.prototype = {
  candles: function (callback, market, granularity) {
    this.api.request('GET', '/markets/' + market + '/candles/' + granularity, undefined, function (resp) {
      return callback(resp);
    });
  }
};

export default Market;
