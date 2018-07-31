import './index.scss';
import './trade.scss';
import $ from 'jquery';
import Chart from './chart.js';

function Home(router, api) {
  this.router = router;
  this.api = api;
  this.templateIndex = require('./index.html');
  this.templateTrade = require('./trade.html');
}

Home.prototype = {
  index: function () {
    const self = this;
    var data = require('./depth.json');
    var bids = data.data.bids;
    var asks = data.data.asks;
    var asksData = [];
    var bidsData = [];
    for (var i = 0; i < bids.length; i++) {
      bidsData.push({
        price: parseFloat(bids[i].price).toFixed(8),
        amount: parseFloat(bids[i].amount).toFixed(4)
      });
    }
    for (var i = asks.length; i > 0; i--) {
      asksData.push({
        price: parseFloat(asks[i-1].price).toFixed(8),
        amount: parseFloat(asks[i-1].amount).toFixed(4)
      });
    }

    $('body').attr('class', 'market layout');
    $('#layout-container').html(self.templateIndex({
      logoURL: require('./logo.png'),
      symbolURL: require('./symbol.png')
    })).append(self.templateTrade({
      asks: asksData,
      bids: bidsData
    }));
    $('.market.detail.spacer').height($('.market.detail.container').outerHeight());
    $('.market.detail.container').addClass('fixed');
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
      if (scroll < height - $('.market.detail .header.container').outerHeight()) {
        $('.market.detail.container').removeClass('visible');
      } else {
        $('.market.detail.container').addClass('visible');
      }
      if (scroll < height - 4) {
        if (scroll < height - $(window).height() * 2 / 3 && $(window).width() > 1200) {
          $('.markets.nav').fadeIn();
        }
        $('.market.detail.spacer').show();
        $('.market.detail.container').addClass('fixed');
      } else if (scroll > height + 4){
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

    var total = $('.order.book').height() - $('.order.book .spread').outerHeight();
    var count = parseInt(total / $('.order.book .ask').outerHeight() / 2) * 2;
    var line = (total / count) + 'px';
    $('.order.book .ask').css({'line-height': line, height: line});
    $('.order.book .bid').css({'line-height': line, height: line});
    $('.order.book .header').css({'line-height': line, height: line});

    var chart = new Chart();
    chart.renderPrice($('.price.chart')[0]);
    chart.renderDepth($('.depth.chart')[0], bids, asks);
    self.api.subscribe('c94ac88f-4671-3976-b60a-09064f1811e8-c6d0c728-2624-429b-8e0d-d9d19b6592fa', self.render);
  },

  render: function (msg) {
    console.log(msg);
  }
};

export default Home;
