$(document).ready(init);

function init() {
    $('body').on('click', 'td', fnAjaxLoadPage);
    //$("#iconTableUser").click(fnAjaxLoadPage);
    fnLog("Loaded");
}

function fnAjaxLoadPage(e) {
    fnLog("Clicked" + e.target.rowIndex);
    jQuery.ajax(
        {
            type	: 'post',
            url		: "/ajax",
            data	: 
            {
                state : "admin-home"
            },
            dataType: 'json',
            success: function(result) {
                fnLog("Success:"+result);
            },
            error: function(result) {
                fnLog("Failure:"+result);
            }
        });
}

function fnNum2ZPfxdStr(num, requiredLength) {
    var numStr = num.toString();
    var lenNumStr = numStr.length;
    var diffLen = requiredLength - lenNumStr;
    for (var i = 0; i < diffLen; i++)
    {
       numStr += '0';
    }
    /*
    switch (diff_len) {
        case 1:
            num_str = '0' + num_str;
            break;
        case 2:
            num_str = '00' + num_str;
            break;
        case 3:
            num_str = '000' + num_str;
            break;
        default:
            break;
    }
    */
    return numStr;
}

function fnLog(logStr) {
    var dt = new Date();

    var dateTimeStamp = dt.getFullYear() + "/" + fnNum2ZPfxdStr(dt.getMonth() + 1, 2) + "/" + fnNum2ZPfxdStr(dt.getDate(), 2) + " " + fnNum2ZPfxdStr(dt.getHours(), 2) + ":" + fnNum2ZPfxdStr(dt.getMinutes(), 2) + ":" + fnNum2ZPfxdStr(dt.getSeconds(), 2) + "." + fnNum2ZPfxdStr(dt.getMilliseconds(), 3);
    //var date_str = Date($.now());
    console.log("@" + dateTimeStamp + "> " + logStr);
}