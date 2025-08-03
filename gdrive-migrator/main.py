import argparse
import io
import json
import os
from pathlib import Path

from google.oauth2 import service_account
from googleapiclient.discovery import build
from googleapiclient.http import MediaIoBaseDownload
from PIL import Image
from pillow_heif import register_heif_opener
from tqdm import tqdm

# Initialize HEIF support for Pillow
register_heif_opener()

class GoogleDriveHEICConverter:
    def __init__(self, output_folder, target_format="JPEG", quality=100, subsampling=0):
        self.SCOPES = ['https://www.googleapis.com/auth/drive.readonly']
        self.output_folder = Path(output_folder)
        self.target_format = target_format.upper()
        self.quality = quality  # 1-100 for JPEG/WEBP, compression level for PNG
        self.subsampling = subsampling  # 0=best, 1=medium, 2=worst quality
        self.service = self.authenticate_with_service_account()

    def authenticate_with_service_account(self):
        creds_file = os.getenv('GOOGLE_APPLICATION_CREDENTIALS')
        if not creds_file:
            raise ValueError("GOOGLE_APPLICATION_CREDENTIALS environment variable not set")
        
        if not os.path.exists(creds_file):
            raise FileNotFoundError(f"Credentials file not found at: {creds_file}")
        
        with open(creds_file) as f:
            creds_info = json.load(f)
        
        creds = service_account.Credentials.from_service_account_info(
            creds_info,
            scopes=self.SCOPES
        )
        return build('drive', 'v3', credentials=creds)

    def get_heic_files(self, folder_id):
        query = (
            f"'{folder_id}' in parents and "
            "(mimeType='image/heic' or "
            "mimeType='image/heif' or "
            "name contains '.heic' or "
            "name contains '.HEIC')"
        )
        results = self.service.files().list(
            q=query,
            pageSize=1000,
            fields="nextPageToken, files(id, name, mimeType)"
        ).execute()
        return results.get('files', [])

    def download_file(self, file_id, file_name):
        request = self.service.files().get_media(fileId=file_id)
        fh = io.BytesIO()
        downloader = MediaIoBaseDownload(fh, request)
        
        with tqdm(desc=f"Downloading {file_name}", unit='B', unit_scale=True) as pbar:
            done = False
            while not done:
                status, done = downloader.next_chunk()
                if status:
                    pbar.update(status.resumable_progress - pbar.n)
        
        fh.seek(0)
        return fh

    def convert_image(self, file_stream, original_name):
        try:
            self.output_folder.mkdir(parents=True, exist_ok=True)
            output_name = original_name.rsplit('.', 1)[0] + f'.{self.target_format.lower()}'
            output_path = self.output_folder / output_name
            
            img = Image.open(file_stream)
            
            # Convert to RGB if needed (JPEG doesn't support alpha channel)
            if img.mode in ('RGBA', 'LA', 'P'):
                img = img.convert('RGB')
            elif img.mode == 'I;16':
                img = img.convert('I')
            
            save_kwargs = {
                'format': self.target_format,
                'quality': self.quality,
            }
            
            # Format-specific optimization
            if self.target_format == 'JPEG':
                save_kwargs.update({
                    'subsampling': self.subsampling,  # 0 for best quality
                    'optimize': True,
                    'progressive': False,  # Baseline JPEG for maximum compatibility
                })
            elif self.target_format == 'PNG':
                save_kwargs.update({
                    'compress_level': 9 - round((self.quality / 100) * 9),  # Convert 0-100 to 0-9
                })
            elif self.target_format == 'WEBP':
                save_kwargs.update({
                    'method': 6,  # 0=fast, 6=best quality
                    'lossless': False,
                })
            
            img.save(output_path, **save_kwargs)
            
            return True, output_path
        except Exception as e:
            return False, str(e)

    def process_folder(self, folder_id):
        """Process all HEIC/HEIF files in a Google Drive folder"""
        files = self.get_heic_files(folder_id)
        
        if not files:
            print("No HEIC/HEIF files found in the specified folder.")
            return
        
        print(f"Found {len(files)} HEIC/HEIF files to process:")
        
        for file in files:
            print(f"\nProcessing: {file['name']}")
            
            try:
                file_stream = self.download_file(file['id'], file['name'])
                success, result = self.convert_image(file_stream, file['name'])
                
                if success:
                    print(f"Successfully converted to: {result}")
                else:
                    print(f"Conversion failed: {result}")
                    break
                    
            except Exception as e:
                print(f"Error processing {file['name']}: {str(e)}")
                break

def main():
    parser = argparse.ArgumentParser(
        description="Convert HEIC/HEIF images from Google Drive to other formats with maximum quality"
    )
    parser.add_argument(
        '--folder-id',
        required=True,
        help='Google Drive folder ID containing HEIC files'
    )
    parser.add_argument(
        '--output',
        default='./converted_photos',
        help='Output directory for converted images (default: ./converted_photos)'
    )
    parser.add_argument(
        '--format',
        default='JPEG',
        choices=['JPEG', 'PNG', 'WEBP'],
        help='Target format for conversion (default: JPEG)'
    )
    parser.add_argument(
        '--quality',
        type=int,
        default=100,
        choices=range(1, 101),
        metavar="[1-100]",
        help='Quality setting (1-100, default: 100)'
    )
    parser.add_argument(
        '--subsampling',
        type=int,
        default=0,
        choices=[0, 1, 2],
        help='JPEG chroma subsampling (0=best, 1=medium, 2=worst quality, default: 0)'
    )
    
    args = parser.parse_args()
    
    converter = GoogleDriveHEICConverter(
        output_folder=args.output,
        target_format=args.format,
        quality=args.quality,
        subsampling=args.subsampling
    )
    
    converter.process_folder(args.folder_id)

if __name__ == '__main__':
    main()