function Ocean(api) {
  this.api = api;
}

Ocean.prototype = {
  orders: function (callback, market, offset) {
    this.api.request('GET', 'https://events.ocean.one/orders?state=PENDING&limit=100&market=' + market + '&offset=' + offset, undefined, function (resp) {
      callback(resp);
    });
  }
};

export default Ocean;
