import { combineReducers } from 'redux';
import commentsReducer from './slices/commentsSlice.js';
import communitiesReducer from './slices/communitiesSlice.js';
import feedsReducer from './slices/feedsSlice.js';
import mainReducer from './slices/mainSlice.js';
import postsReducer from './slices/postsSlice.js';
import usersReducer from './slices/usersSlice.js';
import listsReducer from './slices/listsSlice.js';

const rootReducer = combineReducers({
  main: mainReducer,
  posts: postsReducer,
  feeds: feedsReducer,
  comments: commentsReducer,
  communities: communitiesReducer,
  users: usersReducer,
  lists: listsReducer,
});

export default rootReducer;
