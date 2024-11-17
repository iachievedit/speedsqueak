import zmq
import json
import time
import structlog
import sys
import logging
import threading
from picamera2 import Picamera2
from PIL import Image, ImageOps, ImageFont, ImageDraw

ZMQ_SUBSCRIBE_PORT = 11205
ZMQ_PUBLISH_PORT   = 11206

def standard_fields(_, __, dict):
  dict["service"] = "speedsqueak-camera"
  return dict

structlog.configure(
    processors=[
        standard_fields,
        structlog.processors.add_log_level,
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.JSONRenderer()         
    ],
    wrapper_class=structlog.make_filtering_bound_logger(logging.INFO),
    context_class=dict,
    logger_factory=structlog.stdlib.LoggerFactory(),
)

logger = structlog.get_logger()

file_handler = logging.FileHandler("/var/log/speedsqueak/camera.log")
file_handler.setFormatter(logging.Formatter('%(message)s'))
root_logger = logging.getLogger()
root_logger.setLevel(logging.INFO)
root_logger.addHandler(file_handler)

def main():

  logger.info("speedsqueak-camera booting")

  try:
    picam2 = Picamera2()
    picam2.preview_configuration.main.size = (1920,1080)
    picam2.configure("preview")
    picam2.start()
  except:
    logger.fatal("Failed to initialize the camera")
    sys.exit(-1)

  context = zmq.Context()
  topic = "event/speed"
  subscriber = context.socket(zmq.SUB)
  subscriber.connect(f"tcp://localhost:{ZMQ_SUBSCRIBE_PORT}")
  subscriber.setsockopt_string(zmq.SUBSCRIBE, topic)

  publisher = context.socket(zmq.PUB)
  publisher.bind(f"tcp://localhost:{ZMQ_PUBLISH_PORT}")

  logger.info("Subscribed to topic", topic=topic)

  try:
    while True:
        # Receive the message
        message = subscriber.recv_string()

        # Split the topic and the actual JSON payload
        topic, json_data = message.split(" ", 1)

        # Parse the JSON payload
        data = json.loads(json_data)
        uuid      = data['uuid']
        timestamp = data['timestamp']
        reading   = data['reading']

        logger.info("Received message on topic", topic=topic, data=data)

        image = picam2.capture_array()
        logger.info("Image captured", uuid=uuid)

        img = Image.fromarray(image).convert("RGB")
        #img = ImageOps.flip(img)
        #img = ImageOps.mirror(img)

        draw = ImageDraw.Draw(img)
        font = ImageFont.truetype("/usr/share/fonts/truetype/noto/NotoSansMono-Regular.ttf", 48)

        timestamp_text = timestamp
        speed_text = f"{reading} mph"

        width, height = img.size
        padding = 10
        left_position = (padding, height - font.getsize(timestamp_text)[1] - padding)
        right_position = (width - draw.textsize(speed_text, font=font)[0] - padding, height - font.getsize(speed_text)[1] - padding)

        draw.text(left_position, timestamp_text, font=font, fill="white")
        draw.text(right_position, speed_text, font=font, fill="white")

        filename = f"./images/capture_{uuid}.jpg"
        try:
          img.save(filename)
          logger.info("Image saved", filename=filename)
        except:
          logger.error("Failed to save image")
                
        
        event = {
          'type':      'camera',
          'uuid':      uuid,
          'filepath':  filename,
        }

        publisher.send_string("event/camera %s" % json.dumps(event))
        logger.info("Published message", topic="event/camera")
    
  
  except KeyboardInterrupt:
    print("\nShutting down subscriber...")

  finally:
    subscriber.close()
    context.term()

if __name__ == "__main__":
    main()
