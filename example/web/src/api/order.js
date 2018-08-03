function Order(api) {
  this.api = api;
}

Order.prototype = {
  create: function (callback, params) {
    this.api.request('POST', '/orders', params, function (resp) {
      callback(resp);
    });
  },

  cancel: function (callback, id) {
    this.api.request('POST', '/orders/' + id + '/cancel', undefined, function (resp) {
      callback(resp);
    });
  }
};

export default Order;
