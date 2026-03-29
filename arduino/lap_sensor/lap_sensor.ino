// Mini 4WD Lap Timer - IR Sensor Sketch
// Sends "TRIGGER\n" over serial (9600 baud) each time the IR beam is broken.
// Wiring: IR receiver OUT pin -> D2, VCC -> 5V, GND -> GND

const int SENSOR_PIN = 2;
const int LED_PIN = 13;
const unsigned long DEBOUNCE_MS = 500;

int lastState = HIGH;
unsigned long lastTriggerTime = 0;

void setup() {
  Serial.begin(9600);
  pinMode(SENSOR_PIN, INPUT_PULLUP);
  pinMode(LED_PIN, OUTPUT);
  digitalWrite(LED_PIN, LOW);
}

void loop() {
  int state = digitalRead(SENSOR_PIN);
  unsigned long now = millis();

  // Falling edge: beam broken by car
  if (state == LOW && lastState == HIGH) {
    if (now - lastTriggerTime > DEBOUNCE_MS) {
      Serial.println("TRIGGER");
      digitalWrite(LED_PIN, HIGH);
      lastTriggerTime = now;
    }
  }

  if (state == HIGH && lastState == LOW) {
    digitalWrite(LED_PIN, LOW);
  }

  lastState = state;
}
