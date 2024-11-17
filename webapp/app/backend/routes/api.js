// Don't make fun of the code, ChatGPT wrote it, which means... you wrote it.

const express = require('express');
const snowflake = require('snowflake-sdk');
const { BlobServiceClient, generateBlobSASQueryParameters, BlobSASPermissions, StorageSharedKeyCredential } = require('@azure/storage-blob');
const router = express.Router();
const dotenv = require('dotenv');

dotenv.config();

console.log(process.env);

const accountName = process.env.AZURE_STORAGE_NAME;
const accountKey = process.env.AZURE_STORAGE_KEY;
const containerName = 'images';

// Generate SAS URL
router.get('/generate-sas-url', async (req, res) => {
  const blobName = req.query.blobName;
  
  if (!blobName) {
    return res.status(400).send('Blob name is required');
  }

  try {
    const sharedKeyCredential = new StorageSharedKeyCredential(accountName, accountKey);
    const blobServiceClient = new BlobServiceClient(`https://${accountName}.blob.core.windows.net`, sharedKeyCredential);
    const blobClient = blobServiceClient.getContainerClient(containerName).getBlobClient(blobName);

    const sasToken = generateBlobSASQueryParameters({
      containerName,
      blobName,
      expiresOn: new Date(new Date().valueOf() + 3600 * 1000), // 1 hour expiry
      permissions: BlobSASPermissions.parse('r'),
    }, sharedKeyCredential).toString();

    res.json({ sasUrl: `${blobClient.url}?${sasToken}` });
  } catch (error) {
    console.error('Error generating SAS URL:', error);
    res.status(500).send('Failed to generate SAS URL');
  }
});


// Snowflake configuration
const snowflakeConnection = snowflake.createConnection({
  account: process.env.SNOWFLAKE_ACCOUNT,
  username: process.env.SNOWFLAKE_USER,
  password: process.env.SNOWFLAKE_PASS,
  warehouse: process.env.SNOWFLAKE_WAREHOUSE,
  database: 'speedsqueak',
  schema: 'public',
});

// Test Snowflake Connection
snowflakeConnection.connect((err) => {
  if (err) {
    console.error('Snowflake connection failed:', err);
  } else {
    console.log('Connected to Snowflake');
  }
});


router.get('/events', (req, res) => {
  const query = 'SELECT * FROM events ORDER BY timestamp DESC';

  snowflakeConnection.execute({
    sqlText: query,
    complete: (err, stmt, rows) => {
      if (err) {
        console.error('Snowflake query error:', err);
        res.status(500).send('Failed to fetch data from Snowflake');
      } else {
        res.json(rows);
      }
    },
  });
});

module.exports = router;
