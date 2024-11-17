import zmq
import json
import time
import structlog
import logging
import sqlite3
from azure.storage.blob import BlobServiceClient
import platform
import snowflake.connector
from dotenv import load_dotenv
import os

# Load the .env file
load_dotenv()

# Access environment variables
AZURE_CONNECTION_STRING = os.getenv("AZURE_CONNECTION_STRING")
SNOWFLAKE_USER          = os.getenv("SNOWFLAKE_USER")
SNOWFLAKE_PASS          = os.getenv("SNOWFLAKE_PASS")
SNOWFLAKE_ACCOUNT       = os.getenv("SNOWFLAKE_ACCOUNT") 
SNOWFLAKE_WAREHOUSE     = os.getenv("SNOWFLAKE_WAREHOUSE")

def log_path():
  if platform.system() == "Darwin":
    return "/tmp/logs"
  elif platform.system() == "Linux":
    return "/var/log"
  else:
    return "/tmp/logs"

def standard_fields(_, __, dict):
    dict["service"] = "speedsqueak-uploader"
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

log_path = log_path()
file_handler = logging.FileHandler(f"{log_path}/speedsqueak/uploader.log")
file_handler.setFormatter(logging.Formatter('%(message)s'))
root_logger = logging.getLogger()
root_logger.setLevel(logging.INFO)
root_logger.addHandler(file_handler)

AZURE_CONTAINER_NAME = "images"

blob_service_client = BlobServiceClient.from_connection_string(AZURE_CONNECTION_STRING)
container_client = blob_service_client.get_container_client(AZURE_CONTAINER_NAME)

def upload_event_to_blob(uuid, data):
  try:
    blob_name = f"{uuid}.jpg"
    blob_client = container_client.get_blob_client(blob_name)

    image_path = f"/usr/local/src/speedsqueak/camera/{data}"
    
    # Open the image file in binary mode and upload it
    with open(image_path, "rb") as image_file:
      blob_client.upload_blob(image_file, overwrite=True)
    
    logger.info(f"Uploaded {uuid} to blob storage as {blob_name}")

    return True
  except Exception as e:
    logger.error(f"Failed to upload {uuid} to blob storage: {e}")
    return False

conn = sqlite3.connect("events.db")
cursor = conn.cursor()

snowflake_conn = snowflake.connector.connect(
    user=SNOWFLAKE_USER,
    password=SNOWFLAKE_PASS,
    account=SNOWFLAKE_ACCOUNT,
    database='speedsqueak',
    schema='public',
    )
snowflake_conn.cursor().execute(f"USE WAREHOUSE COMPUTE_WH")


def main():

  logger.info("speedsqueak-uploader booting")

  context = zmq.Context()
  subscriber = context.socket(zmq.SUB)
  subscriber.connect("tcp://localhost:11205") # Radar
  subscriber.connect("tcp://localhost:11206") # Camera
  subscriber.connect("tcp://localhost:11207") # Heartbeat
  subscriber.setsockopt_string(zmq.SUBSCRIBE, "event/")

  while True:
    message = subscriber.recv_string()
    topic, data = message.split(" ", 1)

    if topic == "event/speed":

      logger.info("Radar event received")

      event = json.loads(data)
      uuid  = event["uuid"] 

      cursor.execute("""
                     INSERT INTO events (uuid, speed_data, uploaded)
                     VALUES (?, ?, ?)
                     """, (uuid, data, False))

      conn.commit()
    elif topic == "event/camera":
      logger.info("Camera event received")

      event = json.loads(data)
      uuid  = event["uuid"]

      print(event)

      cursor.execute("""
                UPDATE events SET camera_data=? WHERE uuid=?
                """, (data, uuid))
      conn.commit()

    elif topic == "event/heartbeat":
      logger.info("Heartbeat received")

      logger.info("Uploading complete events")

      cursor.execute("""
                    SELECT uuid, 
                     json_extract(speed_data, '$.timestamp') AS timestamp,
                     json_extract(speed_data, '$.reading') AS speed,
                     json_extract(camera_data, '$.filepath') AS filepath
                    FROM events where filepath!='' AND uploaded=false;
                     """)

      rows = cursor.fetchall()

      logger.info("Images to upload", count=len(rows))

      for r in rows:
        uuid      = r[0]
        timestamp = r[1]
        speed     = r[2]
        filepath  = r[3]
        uploaded = upload_event_to_blob(uuid, filepath)

        if uploaded:
          cursor.execute("""
                        UPDATE events SET uploaded=? WHERE uuid=?
                        """, (uploaded, uuid))

          # Update Snowflake
          snowflake_conn.cursor().execute("""
                                          INSERT INTO events (UUID, LOCATION, IMAGE, SPEED, TIMESTAMP) VALUES (%s,%s,%s,%s,%s)""",
                                          (uuid, "speedsqueak3", f"{uuid}.jpg", speed, timestamp))
          
          conn.commit()

    else:
      logger.info(f"Unknown topic {topic}, ignoring")

    
main()