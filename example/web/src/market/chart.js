import $ from 'jquery';
import {BigNumber} from 'bignumber.js';

function Chart() {
  Highcharts.setOptions({
    time: {
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone
    }
  });
}

Chart.prototype = {
  prepareCandleData: function (data) {
    var ohlc = [],
      volume = [],
      dataLength = data.length;

    for (var i = 0; i < dataLength; i += 1) {
      ohlc.push([
        data[i][0] * 1000, // the date
        data[i][3], // open
        data[i][2], // high
        data[i][1], // low
        data[i][4] // close
      ]);

      volume.push([
        parseFloat(new BigNumber(data[i][0]).times(1000).toFixed(8)), // the date
        parseFloat(new BigNumber(data[i][5]).toFixed(8)) // the volume
      ]);
    }

    return [ohlc, volume];
  },

  renderPrice: function (ele, currency, data) {
    var groupingUnits = [
      ['minute', [1, 5, 15, 30]],
      ['hour', [1, 6, 12, 24]]
    ];

    data = this.prepareCandleData(data);
    var ohlc = data[0];
    var volume = data[1];

    var chart = Highcharts.stockChart(ele, {
      chart: {
        zoomType: 'none',
        pinchType: 'none',
        panning: false,
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
          stickyTracking: false,
          showInLegend: false
        }
      },

      yAxis: [{
        labels: {
          align: 'right',
          x: -3,
          formatter: function () {
            return new BigNumber(this.value).toString(10);
          }
        },
        height: '70%',
        gridLineWidth: 0.5,
        lineWidth: 0
      }, {
        labels: {
          align: 'right',
          x: -3
        },
        top: '71%',
        height: '29%',
        offset: 0,
        gridLineWidth: 0.5,
        lineWidth: 0
      }],

      tooltip: {
        followPointer: true,
        followTouchMove: false,
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
        color: 'rgba(41,149,242,0.3)'
      }, {
        type: 'candlestick',
        id: 'candle',
        name: currency,
        data: ohlc,
        dataGrouping: {
          units: groupingUnits
        }
      }, {
        type: 'ema',
        linkedTo: 'candle',
        params: {
          period: 12
        },
        color: 'rgba(255,155,100,0.5)',
        lineWidth: 1
      }, {
        type: 'ema',
        linkedTo: 'candle',
        params: {
          period: 26
        },
        color: 'rgba(100,155,255,0.5)',
        lineWidth: 1
      }]
    });

    return chart;
  },

  renderDepth: function (ele, bids, asks, depth) {
    if (bids.length === 0 || asks.length === 0) {
      return undefined;
    }

    var bidsData = [];
    for(var i = 0; i < bids.length; i++) {
      bids[i].volume = parseFloat(bids[i].amount);
      if (i > 0) {
        bids[i].volume = parseFloat(new BigNumber(bids[i-1].volume).plus(bids[i].volume).toFixed(8));
      }
      bidsData.push({
        x: parseFloat(bids[i].price),
        y: bids[i].volume
      });
    }
    bidsData = bidsData.reverse();
    bidsData = bidsData.splice(bidsData.length * 1 / 4 + bidsData.length * depth / 2);

    var asksInput = [];
    for(var i = 0; i < asks.length; i++) {
      asks[i].volume = parseFloat(parseFloat(asks[i].amount).toFixed(4));
      if (i > 0) {
        asks[i].volume = parseFloat(new BigNumber(asks[i-1].volume).plus(asks[i].volume).toFixed(4));
      }
      asksInput.push({
        x: parseFloat(asks[i].price),
        y: asks[i].volume
      });
    }
    var asksData = [];
    var priceThreshold = bidsData[bidsData.length - 1].x + asksInput[0].x - bidsData[0].x;
    for (var i = 0; i < asksInput.length; i++) {
      var point = asksInput[i];
      if (point.x > priceThreshold && asksData.length > 10) {
        break;
      }
      asksData.push(point);
    }

    var minPrice = bidsData[0].x;
    var maxPrice = asksData[asksData.length-1].x;
    var maxVolume = bidsData[0].y;
    if (asksData[asksData.length-1].y > maxVolume) {
      maxVolume = asksData[asksData.length-1].y;
    }

    var chart = Highcharts.chart(ele, {
      chart: {
        zoomType: 'none',
        pinchType: 'none',
        panning: false,
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
        max: maxPrice,
        labels: {
          formatter: function () {
            return new BigNumber(this.value).toString(10);
          }
        }
      },

      yAxis: {
        opposite: true,
        labels: {
          align: 'right',
          x: -3,
          y: -2,
          formatter: function () {
            return new BigNumber(this.value).toString(10);
          }
        },
        lineWidth: 0,
        resize: {
          enabled: true
        },
        gridLineWidth: 0.5,
        max: maxVolume,
        min: 0,
        title: {
          text: '',
        },
      },

      plotOptions: {
        series: {
          stickyTracking: false,
          animation: false,
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
        followPointer: true,
        followTouchMove: false,
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

    return chart;
  }
};

export default Chart;
