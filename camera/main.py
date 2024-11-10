import zmq
import json
import time
from picamera2 import Picamera2
from PIL import Image

def main():

    picam2 = Picamera2()
    config = picam2.create_preview_configuration()
    picam2.configure(config)
    picam2.start()
    time.sleep(1)

    # Create a ZeroMQ context
    context = zmq.Context()

    # Create a SUB (subscriber) socket
    subscriber = context.socket(zmq.SUB)

    # Connect to the publisher (change the address if needed)
    subscriber.connect("tcp://localhost:11205")

    # Subscribe to the "speed/events" topic
    topic = "speed/events"
    subscriber.setsockopt_string(zmq.SUBSCRIBE, topic)

    print(f"Subscribed to topic: {topic}")

    try:
        while True:
            # Receive the message
            message = subscriber.recv_string()

            # Split the topic and the actual JSON payload
            topic, json_data = message.split(" ", 1)

            # Parse the JSON payload
            data = json.loads(json_data)

            # Process the data (here we just print it)
            print(f"Received message on topic '{topic}': {data}")

            print("Take picture")
            image = picam2.capture_array()
            print("Image captured")
   
            img = Image.fromarray(image).convert("RGB")
            uuid = data['uuid']
            filename = f"capture_{uuid}.jpg"
            img.save(filename)
            print(f"Image saved as {filename}")

    except KeyboardInterrupt:
        print("\nShutting down subscriber...")

    finally:
        subscriber.close()
        context.term()

if __name__ == "__main__":
    main()

