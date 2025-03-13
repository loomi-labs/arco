<script setup lang='ts'>
import { nextTick, onUnmounted, ref, useId, useTemplateRef, watch } from "vue";
import { useRouter } from "vue-router";
import { showAndLogError } from "../common/error";
import { PencilIcon } from "@heroicons/vue/24/solid";
import { useForm } from "vee-validate";
import { toTypedSchema } from "@vee-validate/zod";
import * as zod from "zod";
import { object } from "zod";
import { getRepoType, RepoType, toHumanReadableSize } from "../common/repository";
import { toCreationTimeBadge, toRepoTypeBadge } from "../common/badge";
import { toLongDateString, toRelativeTimeString } from "../common/time";
import ArchivesCard from "../components/ArchivesCard.vue";
import { repoStateChangedEvent } from "../common/events";
import { Anchor, Page } from "../router";
import ConfirmModal from "../components/common/ConfirmModal.vue";
import { useToast } from "vue-toastification";
import * as repoClient from "../../bindings/github.com/loomi-labs/arco/backend/app/repositoryclient";
import * as ent from "../../bindings/github.com/loomi-labs/arco/backend/ent";
import * as state from "../../bindings/github.com/loomi-labs/arco/backend/app/state";
import {Events} from "@wailsio/runtime";

/************
 * Variables
 ************/

const router = useRouter();
const toast = useToast();
const repo = ref<ent.Repository>(ent.Repository.createFrom());
const repoId = parseInt(router.currentRoute.value.params.id as string) ?? 0;
const repoState = ref<state.RepoState>(state.RepoState.createFrom());
const loading = ref(true);
const repoType = ref<RepoType>(RepoType.Local);
const nbrOfArchives = ref<number>(0);
const totalSize = ref<string>("-");
const sizeOnDisk = ref<string>("-");
const lastArchive = ref<ent.Archive | undefined>(undefined);
const failedBackupRun = ref<string | undefined>(undefined);
const isIntegrityCheckEnabled = ref(false);
const deletableBackupProfiles = ref<ent.BackupProfile[]>([]);
const confirmDeleteInput = ref<string>("");

const confirmRemoveModalKey = useId();
const confirmRemoveModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(confirmRemoveModalKey);
const confirmDeleteModalKey = useId();
const confirmDeleteModal = useTemplateRef<InstanceType<typeof ConfirmModal>>(confirmDeleteModalKey);

const cleanupFunctions: (() => void)[] = [];

const nameInputKey = useId();
const nameInput = useTemplateRef<InstanceType<typeof HTMLInputElement>>(nameInputKey);

const { meta, errors, defineField } = useForm({
  validationSchema: toTypedSchema(
    object({
      name: zod.string({ required_error: "Enter a name for this repository" })
        .min(3, { message: "Name must be at least 3 characters long" })
        .max(30, { message: "Name is too long" })
    })
  )
});

const [name, nameAttrs] = defineField("name", { validateOnBlur: false });

/************
 * Functions
 ************/

async function getData() {
  try {
    loading.value = true;

    repo.value = await repoClient.Get(repoId) ?? ent.Repository.createFrom();
    name.value = repo.value.name;

    repoType.value = getRepoType(repo.value.location);
    isIntegrityCheckEnabled.value = !!repo.value.nextIntegrityCheck;

    deletableBackupProfiles.value = (await repoClient.GetBackupProfilesThatHaveOnlyRepo(repoId)).filter(r => r !== null) ?? [];
  } catch (error: any) {
    await showAndLogError("Failed to get repository data", error);
  }
  loading.value = false;
}

async function getRepoState() {
  try {
    repoState.value = await repoClient.GetState(repoId);

    nbrOfArchives.value = await repoClient.GetNbrOfArchives(repoId);

    totalSize.value = toHumanReadableSize(repo.value.statsTotalSize);
    sizeOnDisk.value = toHumanReadableSize(repo.value.statsUniqueCsize);
    failedBackupRun.value = await repoClient.GetLastBackupErrorMsg(repoId);

    const archive = await repoClient.GetLastArchiveByRepoId(repoId) ?? undefined;
    // Only set lastArchive if it has a valid ID (id > 0)
    lastArchive.value = archive && archive.id > 0 ? archive : undefined;
  } catch (error: any) {
    await showAndLogError("Failed to get repository state", error);
  }
}

async function saveName() {
  if (meta.value.valid && name.value !== repo.value.name) {
    try {
      repo.value.name = name.value ?? "";
      await repoClient.Update(repo.value);
    } catch (error: any) {
      await showAndLogError("Failed to save repository name", error);
    }
  }
}

function resizeNameWidth() {
  if (nameInput.value) {
    nameInput.value.style.width = "30px";
    nameInput.value.style.width = `${nameInput.value.scrollWidth}px`;
  }
}

async function saveIntegrityCheckSettings() {
  try {
    const result = await repoClient.SaveIntegrityCheckSettings(repoId, isIntegrityCheckEnabled.value);
    repo.value.nextIntegrityCheck = result?.nextIntegrityCheck;
  } catch (error: any) {
    await showAndLogError("Failed to save integrity check settings", error);
  }
}

async function removeRepo() {
  try {
    await repoClient.Remove(repoId);
    toast.success("Repository removed");
    await router.replace({ path: Page.Dashboard, hash: `#${Anchor.Repositories}` });
  } catch (error: any) {
    await showAndLogError("Failed to remove repository", error);
  }
}

async function deleteRepo() {
  try {
    await repoClient.Delete(repoId);
    toast.success("Repository deleted");
    await router.replace({ path: Page.Dashboard, hash: `#${Anchor.Repositories}` });
  } catch (error: any) {
    await showAndLogError("Failed to delete repository", error);
  }
}

/************
 * Lifecycle
 ************/

getData();
getRepoState();

watch(loading, async () => {
  // Wait for the loading to finish before adjusting the name width
  await nextTick();
  resizeNameWidth();
});

cleanupFunctions.push(Events.On(repoStateChangedEvent(repoId), async () => await getRepoState()));

onUnmounted(() => {
  cleanupFunctions.forEach((cleanup) => cleanup());
});


</script>

<template>
  <div v-if='loading' class='flex items-center justify-center min-h-svh'>
    <div class='loading loading-ring loading-lg'></div>
  </div>
  <div v-else class='container mx-auto text-left pt-10'>
    <!-- Data Section -->
    <div class='flex items-center justify-between mb-4'>
      <!-- Name -->
      <label class='flex items-center gap-2'
             :class='`text-arco-purple-500`'>
        <input :ref='nameInputKey'
               type='text'
               class='text-2xl font-bold bg-transparent border-transparent w-10'
               v-model='name'
               v-bind='nameAttrs'
               @change='saveName'
               @input='resizeNameWidth'
        />
        <PencilIcon class='size-4' />
        <span class='text-error'>{{ errors.name }}</span>
      </label>
    </div>

    <div class='flex flex-col w-full py-6'>
      <div class='flex justify-between'>
        <div>{{ $t("archives") }}</div>
        <div>{{ nbrOfArchives }}</div>
      </div>
      <div class='divider'></div>
      <div class='flex justify-between'>
        <div>{{ $t("location") }}</div>
        <div class='flex items-center gap-4'>
          <span>{{ repo.location }}</span>
          <span :class='toRepoTypeBadge(repoType)'>{{ repoType === RepoType.Local ? $t("local") : $t("remote") }}</span>
        </div>
      </div>
      <div class='divider'></div>
      <div class='flex justify-between'>
        <div>{{ $t("last_backup") }}</div>
        <span v-if='failedBackupRun' class='tooltip tooltip-error' :data-tip='failedBackupRun'>
          <span class='badge badge-outline badge-error'>{{ $t("failed") }}</span>
        </span>
        <span v-else-if='lastArchive' class='tooltip' :data-tip='toLongDateString(lastArchive.createdAt)'>
          <span :class='toCreationTimeBadge(lastArchive?.createdAt)'>{{ toRelativeTimeString(lastArchive.createdAt) }}</span>
        </span>
        <span v-else>-</span>
      </div>
      <div class='divider'></div>
      <div class='flex justify-between'>
        <div>{{ $t("total_size") }}</div>
        <div>{{ totalSize }}</div>
      </div>
      <div class='divider'></div>
      <div class='flex justify-between'>
        <div>{{ $t("size_on_disk") }}</div>
        <div>{{ sizeOnDisk }}</div>
      </div>

      <div class='divider'></div>
      <!--      <div class='flex items-center justify-between mb-4'>-->
      <!--        <TooltipTextIcon text='Integrity checks help you to identify data corruptions of your backups'>-->
      <!--          <h3 class='text-xl font-semibold'>Run integrity checks</h3>-->
      <!--        </TooltipTextIcon>-->
      <!--        <input type='checkbox' class='toggle toggle-secondary self-end' v-model='isIntegrityCheckEnabled'-->
      <!--               @change='saveIntegrityCheckSettings'>-->
      <!--      </div>-->
      <!--      <div class='divider'></div>-->
      <div class='flex justify-end gap-2'>
        <button class='btn btn-outline btn-error'
                @click='confirmRemoveModal?.showModal()'
                :disabled='repoState.status !== state.RepoStatus.RepoStatusIdle'
        >Remove
        </button>
        <button class='btn btn-outline btn-error'
                @click='confirmDeleteModal?.showModal()'
                :disabled='repoState.status !== state.RepoStatus.RepoStatusIdle'
        >Delete permanently
        </button>
      </div>

      <ConfirmModal :ref='confirmRemoveModalKey'
                    title='Remove repository'
                    show-exclamation
                    confirm-text='Remove repository'
                    confirm-class='btn-error'
                    @confirm='removeRepo()'>
        <div class='flex flex-col gap-2'>
          <p>Are you sure you want to remove this repository?</p>
          <p>Removing a repository will not delete any backups stored in it. You can add it back later.</p>
          <p v-if='deletableBackupProfiles.length === 1'>The backup profile <span class='font-semibold'>{{ deletableBackupProfiles[0].name }}</span>
            will also be removed.</p>
          <div v-else-if='deletableBackupProfiles.length > 1'>The following backup profiles will also be removed:
            <ul class='list-disc font-semibold pl-5'>
              <li v-for='profile in deletableBackupProfiles'>{{ profile.name }}</li>
            </ul>
          </div>
        </div>
      </ConfirmModal>
      <ConfirmModal :ref='confirmDeleteModalKey'
                    title='Delete repository'
                    show-exclamation
                    @close='confirmDeleteInput = ""'
      >
        <div class='flex flex-col gap-2'>
          <p>Are you sure you want to delete this repository?</p>
          <p>This action is <span class='font-semibold'>irreversible</span> and will <span class='font-semibold'>delete all backups</span>
            stored in this repository!</p>
          <p v-if='deletableBackupProfiles.length === 1'>The backup profile <span class='font-semibold'>{{ deletableBackupProfiles[0].name }}</span>
            will also be deleted!</p>
          <div v-else-if='deletableBackupProfiles.length > 1'>The following backup profiles will also be deleted:
            <ul class='list-disc font-semibold pl-5'>
              <li v-for='profile in deletableBackupProfiles'>{{ profile.name }}</li>
            </ul>
          </div>
          <p class='pt-2'>Type <span class='italic font-semibold'>{{ repo.name }}</span> to confirm.</p>
          <div class='flex items-center gap-2'>
            <input type='text' class='input input-sm input-bordered' v-model='confirmDeleteInput' />
          </div>
        </div>
        <template v-slot:actionButtons>
          <div class='flex gap-3 pt-5'>
            <button type='button' class='btn btn-sm btn-outline'
                    @click='confirmDeleteInput = ""; confirmDeleteModal?.close()'
            >{{ $t("cancel") }}
            </button>
            <button type='button' class='btn btn-sm btn-error'
                    :disabled='confirmDeleteInput !== repo.name'
                    @click='deleteRepo()'
            >Delete repository
            </button>
          </div>
        </template>
      </ConfirmModal>
    </div>

    <ArchivesCard :repo='repo'
                  :repo-status='repoState.status'
                  :highlight='false'
                  :show-backup-profile-column='true'>
    </ArchivesCard>
  </div>
</template>

<style scoped>

</style>