function Order(api) {
  this.api = api;
}

Order.prototype = {
  create: function (callback, params) {
    this.api.request('POST', '/orders', params, function (resp) {
      return callback(resp);
    });
  },

  cancel: function (callback, id) {
    this.api.request('POST', '/orders/' + id + '/cancel', undefined, function (resp) {
      return callback(resp);
    });
  }
};

export default Order;
