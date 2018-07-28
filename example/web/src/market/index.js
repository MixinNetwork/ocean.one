import './index.scss';
import $ from 'jquery';

function Home(router, api) {
  this.router = router;
  this.api = api;
  this.templateIndex = require('./index.html');
}

Home.prototype = {
  index: function () {
    const self = this;
    $('body').attr('class', 'market layout');
    $('#layout-container').html(self.templateIndex({
      logoURL: require('./logo.png')
    }));
    self.api.subscribe('c94ac88f-4671-3976-b60a-09064f1811e8-c6d0c728-2624-429b-8e0d-d9d19b6592fa', self.render);
  },

  render: function (msg) {
    console.log(msg);
  }
};

export default Home;
