import os
import sys
import requests
import random
import json
import re
from threading import Timer
from dotenv import load_dotenv
import openai  # For OpenAI interaction
import logging
import time

# Load environment variables
load_dotenv()

# Configure logging based on .env variable
LOGGING = os.getenv('LOGGING', '1').strip() == '1'

logger = logging.getLogger(__name__)
if LOGGING:
    logger.setLevel(logging.DEBUG)
    # Create a handler that writes to sys.stderr
    handler = logging.StreamHandler(sys.stderr)
    handler.setLevel(logging.DEBUG)
    # Create a logging format
    formatter = logging.Formatter('[%(levelname)s] [%(asctime)s] [%(name)s] %(message)s')
    handler.setFormatter(formatter)
    # Add the handler to the logger
    logger.addHandler(handler)
else:
    logger.disabled = True

# Validate OpenAI API Key
def validate_openai_api_key():
    openai_api_key = os.getenv('OPENAI_API_KEY', '').strip()
    logger.debug(f"Validating OpenAI API Key.")
    if not openai_api_key:
        logger.error("OPENAI_API_KEY is not set or is empty.")
        sys.exit(1)
    else:
        openai.api_key = openai_api_key
        try:
            # Make a simple API call to test the key
            openai.Engine.list()
            logger.info("OpenAI API key is valid.")
        except openai.error.AuthenticationError:
            logger.error("Invalid OpenAI API key.")
            sys.exit(1)
        except openai.error.OpenAIError as e:
            logger.error(f"An error occurred while validating OpenAI API key: {e}")
            sys.exit(1)

def main():
    logger.info("Starting the monitoring script")
    validate_openai_api_key()
    schedule_monitoring()

def schedule_monitoring():
    interval = 2.0  # Interval in seconds
    def run_monitor():
        try:
            monitor_canvas_notes()
        except Exception as e:
            logger.exception(f"Error occurred during monitoring: {e}")
        finally:
            Timer(interval, run_monitor).start()

    run_monitor()

def monitor_canvas_notes():
    logger.info("Running monitor_canvas_notes")

    # Get environment variables for Canvas server details
    target_server = os.getenv('TARGET_SERVER', '').strip()
    api_key = os.getenv('API_KEY', '').strip()

    # Log the environment variables
    logger.debug(f"TARGET_SERVER: {target_server}")
    logger.debug(f"API_KEY: {api_key}")

    # Ensure the target server includes the schema
    if not target_server.startswith("http://") and not target_server.startswith("https://"):
        target_server = "https://" + target_server

    # If target_server or api_key are not properly set, log and raise an error
    if not target_server or not api_key:
        error_message = "TARGET_SERVER or API_KEY is not set."
        logger.error(error_message)
        raise ValueError(error_message)

    # Define the endpoint to get all canvases and find the one named 'JP-API-TEST'
    canvases_endpoint = f"{target_server}/api/v1/canvases"
    headers = {'Private-Token': api_key}
    try:
        logger.info(f"Requesting canvases from {canvases_endpoint}")
        response = requests.get(canvases_endpoint, headers=headers, timeout=10)
        logger.debug(f"Response Status Code: {response.status_code}")
        logger.debug(f"Response Body: {response.text}")
        response.raise_for_status()
        canvases = response.json()
        logger.debug(f"Retrieved canvases: {canvases}")

        canvas_id = None

        # Find the canvas named 'JP-API-TEST'
        for canvas in canvases:
            logger.debug(f"Checking canvas: {canvas.get('name')}")
            if canvas.get('name') == 'JP-API-TEST':
                canvas_id = canvas.get('id')
                logger.info(f"Found canvas 'JP-API-TEST' with ID: {canvas_id}")
                break

        # If the canvas is not found, log and raise an error
        if not canvas_id:
            error_message = "Canvas 'JP-API-TEST' not found."
            logger.error(error_message)
            raise ValueError(error_message)

        # Monitor the canvas for notes containing '{{ }}'
        monitor_notes_endpoint = f"{target_server}/api/v1/canvases/{canvas_id}/notes"
        logger.info(f"Requesting notes from {monitor_notes_endpoint}")
        response = requests.get(monitor_notes_endpoint, headers=headers, timeout=10)
        logger.debug(f"Response Status Code: {response.status_code}")
        logger.debug(f"Response Body: {response.text}")
        response.raise_for_status()
        notes = response.json()
        logger.debug(f"Retrieved notes: {notes}")

        # Regular expression pattern to find text within '{{ }}'
        pattern = re.compile(r'{{(.*?)}}')

        # Iterate over notes to find those containing '{{ }}'
        for note in notes:
            note_text = note.get('text', '')
            logger.debug(f"Processing note ID: {note.get('id')} with text: {note_text}")

            match = pattern.search(note_text)
            if match:
                instruction_text = match.group(1).strip()
                logger.info(f"Found instruction in note ID {note.get('id')}: {instruction_text}")

                # Update the note immediately to mark it as 'processing' by replacing '}}' with '!!Processing!!'
                updated_text = note_text.replace('}}', '!!Processing!!')
                update_note_endpoint = f"{target_server}/api/v1/canvases/{canvas_id}/notes/{note['id']}"
                update_data = {"text": updated_text}
                logger.info(f"Updating note ID {note['id']} to mark as processing")
                response = requests.patch(update_note_endpoint, json=update_data, headers=headers, timeout=10)
                logger.debug(f"Response Status Code: {response.status_code}")
                logger.debug(f"Response Body: {response.text}")
                response.raise_for_status()

                # Process the instruction
                process_instruction(canvas_id, note['id'], instruction_text)
            else:
                logger.debug(f"No unprocessed instruction found in note ID {note.get('id')}")

    except requests.exceptions.RequestException as e:
        logger.exception("A RequestException occurred: %s", e)
        raise
    except Exception as e:
        logger.exception("An unexpected error occurred: %s", e)
        raise

def process_instruction(canvas_id, note_id, instruction_text):
    logger.info(f"Processing instruction for note ID {note_id}")
    # Get environment variables for Canvas server details
    target_server = os.getenv('TARGET_SERVER', '').strip()
    api_key = os.getenv('API_KEY', '').strip()

    # Log the environment variables
    logger.debug(f"TARGET_SERVER: {target_server}")
    logger.debug(f"API_KEY: {api_key}")

    # Ensure the target server includes the schema
    if not target_server.startswith("http://") and not target_server.startswith("https://"):
        target_server = "https://" + target_server

    headers = {'Private-Token': api_key}

try:
        # OpenAI API key is already set and validated

        # Prepare messages for ChatCompletion
        messages = [
            {"role": "system", "content": (
                "You are an assistant that can generate text or images based on user instructions. "
                "For the following instruction, decide whether to generate text or an image. "
                "If you decide to generate text, respond with a JSON object like this:\n"
                '{"type": "text", "content": "<the text you generated>"}\n'
                "If you decide to generate an image, respond with a JSON object like this:\n"
                '{"type": "image", "content": "<the description of the image to generate>"}\n'
                "Do not include any additional text or explanations in your response."
            )},
            {"role": "user", "content":instruction_text}
        ]

        logger.info("Sending request to OpenAI ChatCompletion")
        gpt_response = openai.ChatCompletion.create(
            model="gpt-4o-mini",
            messages=messages,
            max_tokens=500,
            temperature=0.7
        )
        response_text = gpt_response['choices'][0]['message']['content'].strip()
        logger.debug(f"OpenAI response: {response_text}")

        # Parse the response as JSON
        try:
            response_data = json.loads(response_text)
            logger.debug(f"Parsed OpenAI response: {response_data}")
        except json.JSONDecodeError as e:
            logger.error("Failed to parse OpenAI response as JSON: %s", e)
            raise

        # Get the note details to determine its current location
        note_details_endpoint = f"{target_server}/api/v1/canvases/{canvas_id}/notes/{note_id}"
        logger.info(f"Requesting note details from {note_details_endpoint}")
        response = requests.get(note_details_endpoint, headers=headers, timeout=10)
        logger.debug(f"Response Status Code: {response.status_code}")
        logger.debug(f"Response Body: {response.text}")
        response.raise_for_status()
        note = response.json()
        logger.debug(f"Retrieved note details: {note}")

        if response_data['type'] == 'text':
            ai_response = response_data['content']
            logger.info("Processing text response from OpenAI")

            # Generate a random color for the new note, including transparency set to 80% (CC in hex)
            random_color = "#{:02x}{:02x}{:02x}CC".format(
                random.randint(0, 255),
                random.randint(0, 255),
                random.randint(0, 255)
            )
            logger.debug(f"Generated random color: {random_color}")

            # Prepare data for the new note (AI response)
            new_note_data = {
                "title": "AI Response",
                "text": ai_response,
                "depth": note['depth'] + 1,  # make it appear in front of the original note
                "location": {
                    "x": note['location']['x'] + note['size']['width'] * 0.8,
                    "y": note['location']['y'] + note['size']['height'] * 0.8
                },
                "size": note['size'],
                "scale": note['scale'],
                "background_color": random_color  # Random color with 80% transparency
            }

            # Create the new note as a response
            create_note_endpoint = f"{target_server}/api/v1/canvases/{canvas_id}/notes"
            logger.info(f"Creating new note at {create_note_endpoint}")
            response = requests.post(create_note_endpoint, json=new_note_data, headers=headers, timeout=10)
            logger.debug(f"Response Status Code: {response.status_code}")
            logger.debug(f"Response Body: {response.text}")
            response.raise_for_status()
            logger.info("New note created successfully")

            # Now update the original note to mark as 'done'
            updated_text = note['text'].replace('!!Processing!!', '!! Done !!')
            update_note_endpoint = f"{target_server}/api/v1/canvases/{canvas_id}/notes/{note_id}"
            update_data = {"text": updated_text}
            logger.info(f"Updating note ID {note_id} to mark as done")
            response = requests.patch(update_note_endpoint, json=update_data, headers=headers, timeout=10)
            response.raise_for_status()

        elif response_data['type'] == 'image':
            image_description = response_data['content']
            logger.info("Processing image response from OpenAI")

            # Generate image using OpenAI Image API
            image_response = openai.Image.create(
                prompt=image_description,
                n=1,
                size="512x512"
            )
            image_url = image_response['data'][0]['url']
            logger.debug(f"Generated image URL: {image_url}")

            # Download the image
            image_data = requests.get(image_url).content
            logger.info("Image downloaded successfully")

            # Prepare data for the new image
            new_image_json = {
                "title": "AI Generated Image",
                "depth": note['depth'] + 1,
                "location": {
                    "x": note['location']['x'] + note['size']['width'] * 0.8,
                    "y": note['location']['y'] + note['size']['height'] * 0.8
                },
                "size": note['size'],
                "scale": note['scale'],
                # Additional fields can be added if needed
            }

            # Prepare the multipart/form-data request
            files = {
                'json': (None, json.dumps(new_image_json), 'application/json'),
                'data': ('image.png', image_data, 'image/png')
            }

            create_image_endpoint = f"{target_server}/api/v1/canvases/{canvas_id}/images"
            logger.info(f"Creating new image at {create_image_endpoint}")
            response = requests.post(create_image_endpoint, headers=headers, files=files, timeout=10)
            logger.debug(f"Response Status Code: {response.status_code}")
            logger.debug(f"Response Body: {response.text}")
            response.raise_for_status()
            logger.info("New image created successfully")

            # Now update the original note to mark as 'done'
            updated_text = note['text'].replace('!!Processing!!', '!! Done !!')
            update_note_endpoint = f"{target_server}/api/v1/canvases/{canvas_id}/notes/{note_id}"
            update_data = {"text": updated_text}
            logger.info(f"Updating note ID {note_id} to mark as done")
            response = requests.patch(update_note_endpoint, json=update_data, headers=headers, timeout=10)
            response.raise_for_status()

        else:
            # If the type is unknown, log an error
            error_message = f"Unknown response type from OpenAI: {response_data.get('type')}"
            logger.error(error_message)
            raise ValueError(error_message)

except requests.exceptions.RequestException as e:
        logger.exception("A RequestException occurred: %s", e)
        raise
except openai.error.AuthenticationError:
        logger.error("Invalid OpenAI API key during task execution.")
        raise
except openai.error.OpenAIError as e:
        logger.exception("An OpenAIError occurred: %s", e)
        raise
except json.JSONDecodeError as e:
        logger.exception("A JSONDecodeError occurred: %s", e)
        raise
except Exception as e:
        logger.exception("An unexpected error occurred: %s", e)
        raise

if __name__ == "__main__":
    main()
