const data = {"mobile":"07059987","price":100,"currency":"NGN"};
data.mobile = parseInt(data.mobile);
if (data.mobile.toString().length < 10 || data.mobile.toString().length > 11) {
  console.log({"code":502,"data":{"price":100,"currency":"NGN"}});
}