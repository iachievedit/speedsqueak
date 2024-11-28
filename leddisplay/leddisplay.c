/*
 Original code to drive the large 7-segment displays from Nathan Seidle
 of SparkFun Electronics, who asks that if you use this code to buy him a
 beer if you ever meet.
 
 https://learn.sparkfun.com/tutorials/large-digit-driver-hookup-guide
*/

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>
#include <string.h>
#include <signal.h>
#include <time.h>
#include <math.h>

#include <wiringPi.h>
#include <zmq.h>
#include <jansson.h>

void clearLED();

typedef uint8_t byte;

#define a  1<<0
#define b  1<<6
#define c  1<<5
#define d  1<<4
#define e  1<<3
#define f  1<<1
#define g  1<<2
#define dp 1<<7

// WiringPin
byte segmentClock = 0;
byte segmentLatch = 2;
byte segmentData = 3;

void clearLED_and_exit(int signal) {
  clearLED();
  exit(0);
}

// Given a number, or '-', shifts it out to the display
void postNumber(byte number, bool decimal) {

  byte segments;

  //    -       A
  //   / /     F/B
  //    -       G
  //   / /     E/C
  //    -.     D/DP

  switch (number) {
    case 1: segments = b | c; break;
    case 2: segments = a | b | d | e | g; break;
    case 3: segments = a | b | c | d | g; break;
    case 4: segments = f | g | b | c; break;
    case 5: segments = a | f | g | c | d; break;
    case 6: segments = a | f | g | e | c | d; break;
    case 7: segments = a | b | c; break;
    case 8: segments = a | b | c | d | e | f | g; break;
    case 9: segments = a | b | c | d | f | g; break;
    case 0: segments = a | b | c | d | e | f; break;
    case ' ': segments = 0; break;
    case 'c': segments = g | e | d; break;
    case '-': segments = g; break;
  }

  if (decimal) segments |= dp;

  // Clock these bits out to the drivers
  for (uint8_t x = 0 ; x < 8; x++) {
    digitalWrite(segmentClock, LOW);
    digitalWrite(segmentData, segments & 1 << (7 - x));

    // Data transfers to the register on the rising edge of SRCK
    digitalWrite(segmentClock, HIGH); 
  }
}

void showNumber(float value) {

  int number = abs(value); 
  for (unsigned x = 0 ; x < 2 ; x++) {
    int remainder = number % 10;
    if (!remainder) {
      postNumber(' ', false);
    } else {
      postNumber(remainder, false);
    }
    number /= 10;
  }

  digitalWrite(segmentLatch, LOW);
  digitalWrite(segmentLatch, HIGH);
}

void clearLED() {
  postNumber(' ', false);
  postNumber(' ', false);
  digitalWrite(segmentLatch, LOW);
  digitalWrite(segmentLatch, HIGH);
}

void main() {
  
  wiringPiSetup();
  signal(SIGINT, clearLED_and_exit);
  clearLED();

  void* context = zmq_ctx_new();
  void* subscriber = zmq_socket(context, ZMQ_SUB);
  zmq_connect(subscriber, "tcp://0.0.0.0:11205");
  zmq_setsockopt(subscriber, ZMQ_SUBSCRIBE, "event/speed", 0);

  pinMode(segmentClock, OUTPUT);
  pinMode(segmentData, OUTPUT);
  pinMode(segmentLatch, OUTPUT);

  digitalWrite(segmentClock, LOW);
  digitalWrite(segmentData, LOW);
  digitalWrite(segmentLatch, LOW);

  for (;;) {
    char buffer[512] = {0};
    zmq_recv(subscriber, buffer, sizeof(buffer) - 1, 0);
    char* json_start = strchr(buffer, ' ');
    if (json_start) {
      json_start++;

      json_error_t error;
      json_t* root = json_loads(json_start, 0, &error);
      if (!root) {
        fprintf(stderr, "Error parsing JSON: %s\n", error.text);
        continue;
      }

      float speed = json_real_value(json_object_get(root, "reading"));
      speed = floor(speed);
      printf("Speed: %f\n", speed);

      // Flashing effect
      for (uint8_t i = 0; i < 2; i++) {
        showNumber(speed);
        delay(250);
        clearLED();
        delay(250);
      }
  
      showNumber(speed);
      delay(3000);

      clearLED();

      json_decref(root);
    } 
  }
}