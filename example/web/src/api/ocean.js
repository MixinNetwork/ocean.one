function Ocean(api) {
  this.api = api;
}

Ocean.prototype = {
  orders: function (callback) {
    this.api.request('GET', 'https://events.ocean.one/orders', undefined, function (resp) {
      callback(resp);
    });
  }
};

export default Ocean;
