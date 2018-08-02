import $ from 'jquery';

function FormUtils() {
}

FormUtils.prototype = {
  serialize: function (element) {
    var out = {};
    var data = $(element).serializeArray();
    for(var i = 0; i < data.length; i++){
      var record = data[i];
      out[record.name] = record.value;
    }
    return out;
  }
}

export default new FormUtils();
