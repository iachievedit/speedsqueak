// Don't make fun of the code, ChatGPT wrote it, which means... you wrote it.


const express = require('express');
const apiRoutes = require('./routes/api');
const path = require('path');


const app = express();

app.use(express.static(path.join(__dirname, '../frontend/public')));

// Use routes
app.use('/api', apiRoutes);  

// Serve static frontend files
app.use(express.static('../frontend/public'));

const PORT = process.env.PORT || 3333;
app.listen(PORT, () => {
  console.log(`Backend running on http://localhost:${PORT}`);
});
