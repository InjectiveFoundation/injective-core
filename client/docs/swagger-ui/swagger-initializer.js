window.onload = function() {
  //<editor-fold desc="Changeable Configuration Block">

  // Begin Swagger UI call region
  const ui = SwaggerUIBundle({
    url: "./swagger.yaml",
    dom_id: '#swagger-ui',
    deepLinking: true,
    queryConfigEnabled: false,
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    plugins: [
      SwaggerUIBundle.plugins.DownloadUrl
    ],
    layout: "StandaloneLayout"
  });
  // End Swagger UI call region

  window.ui = ui;

  //</editor-fold>
};
