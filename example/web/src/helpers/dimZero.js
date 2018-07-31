var handlebars = require('handlebars');

module.exports = function(value_str) {
	var res = /0{1,}$/.exec(value_str);
	if (res) {
		value_str = new handlebars.SafeString(res.input.slice(0,res.index) + '<span class="num tail">' + res[0] + '</span>');
	}
	res = /^0.0{1,}/.exec(value_str);
	if (res) {
		value_str = new handlebars.SafeString('<span class="num head">' + res[0] + '</span>' + res.input.slice(res[0].length,value_str.length));
	}
  return value_str;
};
