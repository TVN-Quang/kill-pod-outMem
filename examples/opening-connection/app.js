const { json } = require('body-parser');
const express = require('express');
const app = express();
const port = 3000;
const { MongoClient } = require('mongodb');

// URL kết nối tới MongoDB, đảm bảo thay đổi URL này nếu bạn có MongoDB đang chạy trên địa chỉ khác
const url = process.env.DB_URL || 'mongodb://localhost:27017';
const dbName = 'demoDB';

async function run() {
  const client = new MongoClient(url);

  try {
    await client.connect();
    console.log(`${new Date()}: Connected to MongoDB. Wait 10s`);
    await delay(process.env.DELAY_TIME);

    const db = client.db(dbName);
    const collection = db.collection('users');

    // Dữ liệu cần ghi vào MongoDB
    const user = { name: `John Doe ${new Date()}`, email: 'john.doe@example.com' };

    const result = await collection.insertOne(user);
    console.log(`${new Date()}: User inserted with _id: ${result.insertedId}`);
    return result
  } catch (error) {
    console.error('Error connecting to MongoDB:', error);
  } finally {
    const endTime = new Date();
    console.log(`${endTime}: Ending time`);
    await client.close();
  }
}

app.get('/api/delay', async (req, res) => {
  let result = await run()
  res.send('API /api/deplay is working!' + JSON.stringify(result));
})

app.get('/api/loop', async (req, res) => {
  res.send(`API /api/loop is working! Pod ${process.env.POD_NAME} processed`);
})

// Start server
app.listen(port, () => {
  console.log(`Server running at http://localhost:${port}`);
});

function delay(ms) {
  
  return new Promise((resolve) => setTimeout(resolve, parseInt(ms)));
}