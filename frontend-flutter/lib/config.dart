const String apiBaseUrl = String.fromEnvironment(
  'API_BASE_URL',
  defaultValue: 'http://localhost:3001',
);

const int maxItems = 12;
const double stoppingStd = 0.33;
const int likertMin = 1;
const int likertMax = 9;
const int skipValue = 0;
