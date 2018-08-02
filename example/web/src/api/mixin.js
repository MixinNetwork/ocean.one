function Mixin(api) {
  this.api = api;
}

Mixin.prototype = {
  assets: function (callback) {
    this.api.request('GET', 'https://api.mixin.one/assets', undefined, function (resp) {
      callback(resp);
    });
  }
};

export default Mixin;
