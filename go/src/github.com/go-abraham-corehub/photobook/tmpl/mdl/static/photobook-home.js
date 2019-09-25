$(document).ready(init);

function init() {
    getMenuItems()
}

function getMenuItems() {
    $.ajax(
        {
            type: 'post',
            url: "/ajax",
            data:
            {
                x: 1,
                y: 2,
                job: 'loadMenuItems'
            },
            dataType: 'json',
            success: function (result) {
                renderMenus(result.MenuItemsLeft, result.MenuItemsRight)
            }
        });
}

function fn_num_to_z_pfxd_str(num, required_length) {
    var num_str = num.toString()
    var len_num_str = num_str.length;
    var diff_len = required_length - len_num_str;
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
    return num_str;
}

function renderMenus(menuItemsLeft, menuItemsRight) {
    var s = $("#leftMenu").get()
    $.each(menuItemsLeft, function (index, value) {
        elStr = '<a class="mdl-navigation__link" href="">' + value + '</a>'
        $(elStr).appendTo(s[0]);
    });
    
    var s = $("#rightMenu").get()
    $.each(menuItemsRight, function (index, value) {
        elStr = '<li class="mdl-menu__item">' + value + '</li>'
        $(elStr).appendTo(s[0]);
    });
}

function fn_log(log_str) {
    var dt = new Date();

    var datetimestamp = dt.getFullYear() + "/" + fn_num_to_z_pfxd_str(dt.getMonth() + 1, 2) + "/" + fn_num_to_z_pfxd_str(dt.getDate(), 2) + " " + fn_num_to_z_pfxd_str(dt.getHours(), 2) + ":" + fn_num_to_z_pfxd_str(dt.getMinutes(), 2) + ":" + fn_num_to_z_pfxd_str(dt.getSeconds(), 2) + "." + fn_num_to_z_pfxd_str(dt.getMilliseconds(), 3);
    //var date_str = Date($.now());
    console.log("@" + datetimestamp + "> " + log_str);
}