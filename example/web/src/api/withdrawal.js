function Withdrawal(api) {
  this.api = api;
}

Withdrawal.prototype = {
  create: function (callback, params) {
    this.api.request('POST', '/withdrawals', params, function (resp) {
      callback(resp);
    });
  }
};

export default Withdrawal;
