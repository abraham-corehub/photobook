$(document).ready(init);

function init() {
    //$('body').on('click', 'div.appTable', fnAjaxLoadPage);
    $("#appTable tbody tr").on("click", fnAjaxLoadPage);
    fnLog("Loaded");
}

function fnAjaxLoadPage(e) {
    fnLog(e.target.closest('tr').id + ", " + e.target.nodeName + ", " + e.target.innerHTML);
    cRID = "02"
    switch (e.target.nodeName) {
        case 'I':
            switch (e.target.innerHTML) {
                case "person":
                    cRID = "02"
                    break;
                case "edit":
                    cRID = "03"
                    break;
                case "refresh":
                    cRID = "04"
                    break;
                case "delete":
                    cRID = "05"
                    break;
            }
    }
    fnLog(clientRequest);
    jQuery.ajax({
        type: 'post',
        url: "/ajax",
        data: {
            id: cRID
        },
        dataType: 'json',
        success: function (result) {
            fnLog("Success:" + result);
        },
        error: function (result) {
            fnLog("Failure:" + result);
        }
    });
}

function fnNum2ZPfxdStr(num, requiredLength) {
    var numStr = num.toString();
    var lenNumStr = numStr.length;
    var diffLen = requiredLength - lenNumStr;
    for (var i = 0; i < diffLen; i++) {
        numStr += '0';
    }
    return numStr;
}

function fnLog(logStr) {
    var dt = new Date();

    var dateTimeStamp = dt.getFullYear() + "/" + fnNum2ZPfxdStr(dt.getMonth() + 1, 2) + "/" + fnNum2ZPfxdStr(dt.getDate(), 2) + " " + fnNum2ZPfxdStr(dt.getHours(), 2) + ":" + fnNum2ZPfxdStr(dt.getMinutes(), 2) + ":" + fnNum2ZPfxdStr(dt.getSeconds(), 2) + "." + fnNum2ZPfxdStr(dt.getMilliseconds(), 3);
    //var date_str = Date($.now());
    console.log("@" + dateTimeStamp + "> " + logStr);
}