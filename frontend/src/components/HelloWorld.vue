<script lang='ts' setup>
import { reactive, ref } from "vue";
import { Backup, InitRepo, List, Version, CreateSSHKeyPair} from "../../wailsjs/go/borg/Borg";
import { borg } from "../../wailsjs/go/models";

const data = reactive({
  name: "",
  resultText: "Please enter your the repo name below 👇",
  version: "",
  error: "",
  hasRunningBackup: false,
  authorizedKey: "",
});

const listData = ref<borg.ListResponse>();

function version() {
  Version().then((result) => {
    data.version = result;
  }).catch((error) => {
    data.error = error;
  });
}

async function list() {
  List().then((result) => {
    listData.value = result;
  }).catch((error) => {
    data.error = error;
  });
}

async function backup() {
  try {
    data.hasRunningBackup = true;
    await Backup();
    data.resultText = "Backup completed successfully!";
    data.hasRunningBackup = false;
    await list();
  } catch (error: any) {
    data.error = error.toString();
  } finally {
    data.hasRunningBackup = false;
  }
}

async function createRepo() {
  try {
    await InitRepo(data.name);
  } catch (error: any) {
    data.error = error.toString();
  }
}

async function createSSHKeyPair() {
  try {
    data.error = "";
    data.authorizedKey = await CreateSSHKeyPair();
  } catch (error: any) {
    data.error = error.toString();
  }
}

// version()
list();

</script>

<template>
  <main>
    <div id='result' class='result'>{{ data.resultText }}</div>
    <div id='input' class='input-box'>
      <input id='name' v-model='data.name' autocomplete='off' class='input' type='text' />
      <button class='btn' @click='createRepo'>Create Repo</button>
      <br>
      <br>

      <button class='btn' @click='createSSHKeyPair'>Create SSH keypair</button>
      <div v-if='data.authorizedKey' class='result'>{{ data.authorizedKey }}</div>
      <br>
      <br>

      <button class='btn' @click='backup'>Backup</button>
      <!--      Show if a backup is running-->
      <div v-if='data.hasRunningBackup' class='result'>Backup is running...</div>
      <div id='error' class='result' style='color: red'>{{ data.error }}</div>
      <div id='version' class='result'>{{ data.version }}</div>
      <table v-if='listData'>
        <thead>
        <tr>
          <th>Name</th>
          <th>Age</th>
        </tr>
        </thead>
        <tbody>
        <tr v-for='item in listData.archives'>
          <td>{{ item.name }}</td>
          <td>{{ item.time }}</td>
        </tr>
        </tbody>
      </table>
    </div>
  </main>
</template>

<style scoped>
.result {
  height: 20px;
  line-height: 20px;
  margin: 1.5rem auto;
}

.input-box .btn {
  width: 160px;
  height: 30px;
  line-height: 30px;
  border-radius: 3px;
  border: none;
  margin: 0 0 0 20px;
  padding: 0 8px;
  cursor: pointer;
}

.input-box .btn:hover {
  background-image: linear-gradient(to top, #cfd9df 0%, #e2ebf0 100%);
  color: #333333;
}

.input-box .input {
  border: none;
  border-radius: 3px;
  outline: none;
  height: 30px;
  line-height: 30px;
  padding: 0 10px;
  background-color: rgba(240, 240, 240, 1);
  -webkit-font-smoothing: antialiased;
}

.input-box .input:hover {
  border: none;
  background-color: rgba(255, 255, 255, 1);
}

.input-box .input:focus {
  border: none;
  background-color: rgba(255, 255, 255, 1);
}
</style>
