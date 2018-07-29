import './index.scss';
import './trade.scss';
import $ from 'jquery';

function Home(router, api) {
  this.router = router;
  this.api = api;
  this.templateIndex = require('./index.html');
  this.templateTrade = require('./trade.html');
}

Home.prototype = {
  index: function () {
    const self = this;
    $('body').attr('class', 'market layout');
    $('#layout-container').html(self.templateIndex({
      logoURL: require('./logo.png')
    })).append(self.templateTrade());
    $('.market.detail.spacer').height($('.market.detail.container').outerHeight());
    $(window).scroll(function (event) {
      var scroll = $(window).scrollTop();
      var height = $('.layout.header').outerHeight();
      if (scroll - height > -128) {
        $('.layout.nav').fadeIn();
      } else if (scroll - height < -256) {
        $('.layout.nav').fadeOut();
      }

      height = $('.layout.header').outerHeight() + $('.markets.container').outerHeight();
      $('.market.detail.spacer').height($('.market.detail.container').outerHeight());
      if (scroll > height) {
        $('.markets.nav').hide();
        $('.market.detail.spacer').hide();
        $('.market.detail.container').removeClass('fixed');
      } else if (scroll < height){
        if ($(window).width() > 1200) {
          $('.markets.nav').show();
        }
        $('.market.detail.spacer').show();
        $('.market.detail.container').addClass('fixed');
      }
    });
    $('.layout.nav .logo a').click(function() {
      window.scroll({
        top: $('.layout.header').outerHeight() - 128,
        behavior: 'smooth'
      });
    });
    self.api.subscribe('c94ac88f-4671-3976-b60a-09064f1811e8-c6d0c728-2624-429b-8e0d-d9d19b6592fa', self.render);
  },

  render: function (msg) {
    console.log(msg);
  }
};

export default Home;
