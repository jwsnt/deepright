const data = {"mobile":"12345678901","price":100,"currency":"NGN"};

try {
  data.mobile = parseInt(data.mobile);
  if (data.mobile.toString().length < 10 || data.mobile.toString().length > 11) {
    throw new Error('Mobile was error');
  }
  console.log(JSON.stringify(data));
} catch (error) {
  console.error(error.message);
}