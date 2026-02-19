// policy_db/init-scripts/01-create-policyuser.js
const dbName = process.env.MONGO_INITDB_DATABASE;
const username = process.env.POLICY_MONGO_USER;
const password = process.env.POLICY_MONGO_PASSWORD;

if (!username || !password || !dbName) {
  throw new Error(
    "Missing POLICY_MONGO_USER or POLICY_MONGO_PASSWORD or MONGO_INITDB_DATABASE",
  );
}

const targetDb = db.getSiblingDB(dbName);

const existing = targetDb.getUser(username);
if (!existing) {
  targetDb.createUser({
    user: username,
    pwd: password,
    roles: [{ role: "readWrite", db: dbName }],
  });
  print(`Created user '${username}' in db '${dbName}'`);
} else {
  print(`User '${username}' already exists in db '${dbName}'`);
}
