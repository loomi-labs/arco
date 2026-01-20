<script setup lang='ts'>
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import ConnectRepo from "../components/ConnectRepo.vue";
import { Page, withId } from "../router";
import * as backupProfileService from "../../bindings/github.com/loomi-labs/arco/backend/app/backup_profile/service";
import { showAndLogError } from "../common/logger";
import type { Repository } from "../../bindings/github.com/loomi-labs/arco/backend/app/repository";

/************
 * Types
 ************/

/************
 * Variables
 ************/

const router = useRouter();
const route = useRoute();

// Get the source backup profile ID from query params (if coming from backup profile page)
const fromBackupProfileId = computed(() => {
  const id = route.query.fromBackupProfile;
  return id ? parseInt(id as string) : undefined;
});

/************
 * Functions
 ************/

async function handleRepoCreated(repo: Repository) {
  // If coming from a backup profile, add the repo to that profile and return there
  if (fromBackupProfileId.value) {
    try {
      await backupProfileService.AddRepositoryToBackupProfile(fromBackupProfileId.value, repo.id);
      // Note: Toast already shown by the create modal, so we don't add another one here
    } catch (error: unknown) {
      await showAndLogError("Failed to add repository to backup profile", error);
    }
    await router.push(withId(Page.BackupProfile, fromBackupProfileId.value));
  } else {
    // Default behavior: go to the repository page
    await router.push(withId(Page.Repository, repo.id));
  }
}

/************
 * Lifecycle
 ************/

</script>

<template>
  <div class='container mx-auto text-left flex flex-col max-w-[800px]'>

    <h1 class='text-4xl font-bold text-center p-10'>New Repository</h1>

    <ConnectRepo
      :show-add-repo='true'
      @update:repo-added='handleRepoCreated'>
    </ConnectRepo>
  </div>
</template>

<style scoped>

</style>
