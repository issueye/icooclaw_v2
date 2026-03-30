import { createCrudApi } from "./common-api";

const api = createCrudApi("channels");

export const getChannelsPage = api.getPage;
export const getChannels = api.getAll;
export const getEnabledChannels = api.getEnabled;
export const getChannel = api.getById;
export const createChannel = api.create;
export const updateChannel = api.update;
export const deleteChannel = api.remove;
