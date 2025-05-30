// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT

/**
 * ValidationClient is a client for validation related operations
 * @module
 */

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unused imports
import {Call as $Call, Create as $Create} from "@wailsio/runtime";

/**
 * ArchiveName validates the name of an archive.
 * The rules are not enforced by the database because we import them from borg repositories which have different rules.
 */
export function ArchiveName(archiveId: number, prefix: string, name: string): Promise<string> & { cancel(): void } {
    let $resultPromise = $Call.ByID(2656012445, archiveId, prefix, name) as any;
    return $resultPromise;
}

/**
 * RepoName validates the name of a repository.
 * The rules are enforced by the database.
 */
export function RepoName(name: string): Promise<string> & { cancel(): void } {
    let $resultPromise = $Call.ByID(83527431, name) as any;
    return $resultPromise;
}

export function RepoPath(path: string, isLocal: boolean): Promise<string> & { cancel(): void } {
    let $resultPromise = $Call.ByID(371705835, path, isLocal) as any;
    return $resultPromise;
}
