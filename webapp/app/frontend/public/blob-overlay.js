function openOverlay(blobName) {
  fetch(`/api/generate-sas-url?blobName=${blobName}`)
    .then(response => response.json())
    .then(data => {
      const overlay = document.getElementById('blobOverlay');
      const image = document.getElementById('blobImage');
      image.src = data.sasUrl;
      overlay.style.display = 'flex';
    })
    .catch(err => console.error('Error fetching SAS URL:', err));
}

function closeOverlay() {
  const overlay = document.getElementById('blobOverlay');
  const image = document.getElementById('blobImage');
  image.src = '';
  overlay.style.display = 'none';
}
