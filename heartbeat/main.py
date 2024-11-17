#!/usr/bin/env python3
import zmq
import time

context = zmq.Context()
socket = context.socket(zmq.PUB)
socket.bind("tcp://localhost:11207")

topic = "event/heartbeat"
event_data = "Welcome to Costco, I love you."

while True:
  socket.send_string("%s %s" % (topic, event_data))
  time.sleep(60)

