import $ from 'jquery';

function Chart() {
}

Chart.prototype = {
  renderPrice: function (ele) {
    var data = require('./appl-ohlcv.json');
    var ohlc = [],
      volume = [],
      dataLength = data.length,
      groupingUnits = [
        ['minute', [1, 5, 15, 30]],
        ['hour', [1, 6, 12, 24]]
      ];

    for (var i = 0; i < dataLength; i += 1) {
      ohlc.push([
        data[i][0], // the date
        data[i][1], // open
        data[i][2], // high
        data[i][3], // low
        data[i][4] // close
      ]);

      volume.push([
        data[i][0], // the date
        data[i][5] // the volume
      ]);
    }

    Highcharts.stockChart(ele, {
      chart: {
        spacing: [0, 0, 0, 0]
      },

      credits: {
        enabled: false
      },

      rangeSelector: {
        enabled: false
      },

      scrollbar: {
        enabled: false
      },

      navigator: {
        enabled: false
      },

      legend: {
        enabled: false
      },

      plotOptions: {
        series: {
          showInLegend: false
        }
      },

      yAxis: [{
        labels: {
          align: 'right',
          x: -3
        },
        height: '70%',
        lineWidth: 2
      }, {
        labels: {
          align: 'right',
          x: -3
        },
        top: '71%',
        height: '29%',
        offset: 0,
        lineWidth: 2
      }],

      tooltip: {
        split: true
      },

      series: [{
        type: 'column',
        name: 'Volume',
        data: volume,
        yAxis: 1,
        dataGrouping: {
          units: groupingUnits
        },
        color: 'rgba(0,0,0,0.2)'
      }, {
        type: 'candlestick',
        id: 'aapl',
        name: 'AAPL',
        data: ohlc,
        dataGrouping: {
          units: groupingUnits
        }
      }, {
        type: 'ema',
        linkedTo: 'aapl',
        params: {
          period: 12
        },
        color: 'rgba(255,155,100,0.5)',
        lineWidth: 1
      }, {
        type: 'ema',
        linkedTo: 'aapl',
        params: {
          period: 26
        },
        color: 'rgba(100,155,255,0.5)',
        lineWidth: 1
      }]
    });
  },

  renderDepth: function (ele) {
    var data = require('./depth.json');
    var bids = data.data.bids;
    var asks = data.data.asks;

    var bidsData = [];
    for(var i = 0; i < bids.length; i++) {
      bids[i].volume = parseFloat(bids[i].amount);
      if (i > 0) {
        bids[i].volume = bids[i-1].volume + bids[i].volume;
      }
      bidsData.push({
        x: parseFloat(bids[i].price),
        y: bids[i].volume
      });
    }
    bidsData = bidsData.reverse();

    var asksData = [];
    for(var i = 0; i < asks.length; i++) {
      asks[i].volume = parseFloat(asks[i].amount);
      if (i > 0) {
        asks[i].volume = asks[i-1].volume + asks[i].volume;
      }
      asksData.push({
        x: parseFloat(asks[i].price),
        y: asks[i].volume
      });
    }

    var minPrice = bidsData[0].x;
    var maxPrice = asksData[asksData.length-1].x;
    var maxVolume = bidsData[0].y;
    if (asksData[asksData.length-1].y > maxVolume) {
      maxVolume = asksData[asksData.length-1].y;
    }

    var chart = Highcharts.chart(ele, {
      chart: {
        spacing: [0, 0, 0, 0]
      },

      credits: {
        enabled: false
      },

      rangeSelector: {
        enabled: false
      },

      scrollbar: {
        enabled: false
      },

      navigator: {
        enabled: false
      },

      legend: {
        enabled: false
      },

      title: {
        text: null
      },

      xAxis: {
        gridLineWidth: 0.5,
        min: minPrice,
        max: maxPrice
      },

      yAxis: {
        opposite: true,
        labels: {
          align: 'right',
          x: -3,
          y: -2
        },
        lineWidth: 2,
        resize: {
          enabled: true
        },
        gridLineWidth: 0.5,
        max: maxVolume,
        title: {
          text: '',
        },
      },

      plotOptions: {
        series: {
          lineWidth: 1,
          animation: true,
          marker: {
            enabled: false,
            symbol: 'circle',
            states: {
              hover: {
                enabled: true
              }
            }
          }
        }
      },
      tooltip: {
        crosshairs: [true, true],
        formatter: function () {
          return 'Price <b>' + this.x + '</b> <br/>' + this.series.name + '<b>' + this.y + '</b>';
        }
      },
      series: [
        {
          type: 'area',
          name: 'Buy orders',
          data: bidsData,
          color: 'rgba(1,170,120,1.0)',
          fillColor: 'rgba(1,170,120,0.2)'
        },
        {
          type: 'area',
          name: 'Sell orders',
          data: asksData,
          color: 'rgba(255,95,115,1.0)',
          fillColor: 'rgba(255,95,115,0.2)'
        }
      ]
    });

  }
};

export default Chart;
