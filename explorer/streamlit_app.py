import json
import os

import streamlit as st
from st_files_connection import FilesConnection

st.title("Travel Post Viewer")

conn = st.connection("gcs", type=FilesConnection)

post_id = st.selectbox(
    "Select post ID:",
    ["1h8mp0e", "1h9i7j2", "1h9jxmg", "1h9p87g", "1h9td15", "1h9vcf9", "1l48vzb"],
)

if st.button("Load Post"):
    try:
        bucket_path = os.getenv("bucket_path")
        metadata_path = f"{bucket_path}{post_id}/metadata.json"
        metadata = conn.read(metadata_path, input_format="json", ttl=600)

        filtered_metadata = {
            "title": metadata.get("title", ""),
            "text": metadata.get("text", ""),
            "link": metadata.get("reddit_link", ""),
        }

        st.subheader("Post Metadata")
        st.json(filtered_metadata)

        image_patterns = [
            f"{bucket_path}{post_id}/*.jpg",
            f"{bucket_path}{post_id}/*.jpeg",
            f"{bucket_path}{post_id}/*.png",
        ]

        st.subheader("Post Images")
        for pattern in image_patterns:
            try:
                images = conn.fs.glob(pattern)
                for img_path in images:
                    with conn.fs.open(img_path, "rb") as f:
                        image_bytes = f.read()
                        st.image(image_bytes, caption=img_path.split("/")[-1])
            except Exception as e:
                st.warning(f"No images found matching pattern {pattern}")

    except Exception as e:
        st.error(f"Error loading post: {str(e)}")

with st.sidebar:
    st.markdown(
        """
    ### How to use
    1. Select a post ID from the dropdown
    2. Click "Load Post" to view the content
    3. The metadata and images will be displayed below
    
    Note: Content is cached for 600 seconds (10 minutes)
    """
    )
