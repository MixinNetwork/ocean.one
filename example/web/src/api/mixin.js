function Mixin(api) {
  this.api = api;
}

Mixin.prototype = {
  assets: function (callback) {
    this.api.request('GET', 'https://api.mixin.one/assets', undefined, function (resp) {
      return callback(resp);
    });
  },

  asset: function (callback, id) {
    this.api.request('GET', 'https://api.mixin.one/assets/' + id, undefined, function (resp) {
      return callback(resp);
    });
  }
};

export default Mixin;
