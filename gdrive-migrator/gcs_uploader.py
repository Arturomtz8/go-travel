import os

from google.cloud import storage

extensions = [
    '.jpg', '.jpeg', '.png', '.webp',
    '.heic', '.heif', '.bmp', '.tiff', '.gif'
]

def upload_images_to_bucket(namelocalfolder, bucket_name):
    storage_client = storage.Client()
    bucket = storage_client.bucket(bucket_name)
    image_files = [f for f in os.listdir(namelocalfolder) 
                  if os.path.splitext(f)[1].lower() in extensions]
    
    if not image_files:
        print(f"No image files found in {namelocalfolder}")
        return
    
    print(f"Found {len(image_files)} image(s) to upload...")
    
    for image_file in image_files:
        local_path = os.path.join(namelocalfolder, image_file)
        destination_path = f"mine/{namelocalfolder}/{image_file}"
        
        try:
            blob = bucket.blob(destination_path)
            blob.upload_from_filename(local_path)
            print(f"Uploaded {image_file} to {bucket_name}/{destination_path}")
        except Exception as e:
            print(f"Failed to upload {image_file}: {str(e)}")

    
    print("Upload process completed.")

if __name__ == "__main__":
    BUCKET_NAME = os.getenv("GoTravelBucketName")       
    NAME_LOCAL_FOLDER = "athens"
    upload_images_to_bucket(NAME_LOCAL_FOLDER, BUCKET_NAME)
