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
      logoURL: require('./logo.png'),
      symbolURL: require('./symbol.png')
    })).append(self.templateTrade({
      chartURL: require('./chart.png')
    }));
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
      if (scroll > height - $(window).height() * 2 / 3) {
        $('.markets.nav').fadeOut();
        $('.layout.nav .title').html('BTC-USDT')
      } else {
        $('.layout.nav .title').html('USDT MARKETS')
      }
      if (scroll < height) {
        $('.market.detail.container').css({'z-index': -1});
      } else {
        $('.market.detail.container').css({'z-index': 1});
      }
      if (scroll < height) {
        if (scroll < height - $(window).height() * 2 / 3 && $(window).width() > 1200) {
          $('.markets.nav').fadeIn();
        }
        $('.market.detail.spacer').show();
        $('.market.detail.container').addClass('fixed');
      } else if (scroll > height + 16){
        $('.market.detail.spacer').hide();
        $('.market.detail.container').removeClass('fixed');
      }
      if (scroll < 256) {
        $('.market.detail.container').addClass('hidden');
      } else {
        $('.market.detail.container').removeClass('hidden');
      }
    });
    $('.layout.nav .logo a').click(function() {
      window.scroll({
        top: $('.layout.header').outerHeight() - $('.layout.nav').outerHeight(),
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
