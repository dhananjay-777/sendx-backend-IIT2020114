document.addEventListener("DOMContentLoaded", function () {
  const form = document.querySelector("form");
  const resultDiv = document.getElementById("result");

  form.addEventListener("submit", function (e) {
    e.preventDefault(); // Prevent the default form submission behavior

    const urlInput = document.getElementById("urlInput");
    const userStatusSelect = document.getElementById("userStatus");

    const url = urlInput.value;
    const userStatus = userStatusSelect.value;

    // Construct the URL with query parameters
    const urlWithParameters = `/?urlInput=${url}&userStatus=${userStatus}`;
    console.log(urlWithParameters);
    fetch(urlWithParameters, {
      method: "GET",
    })
      .then((response) => response.json())
      .then((data) => {
        resultDiv.textContent = `URL: ${data.url}, User Status: ${data.userStatus}`;
      })
      .catch((error) => {
        console.error("Error:", error);
      });
  });
});
