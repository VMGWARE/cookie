/*
const node = {
  parent: null,
  noRepliesRendered: 0,
  comment: {},
  children: [], // array of nodes
}
// root is just a node
*/

export function searchTree(root, commentId, commentDepth) {
  if (root.comment && root.comment.id === commentId) {
    return root;
  }
  if (!root.children || root.children.length === 0) {
    return null;
  }
  if (
    commentDepth !== undefined &&
    root.children[0].comment.depth > commentDepth
  ) {
    return null;
  }
  for (const node of root.children) {
    if (node.comment.id === commentId) {
      return node;
    }
    if (node.children) {
      const hit = searchTree(node, commentId, commentDepth);
      if (hit) {
        return hit;
      }
    }
  }
  return null;
}

export function Node(comment, children = [], parent = null) {
  return {
    parent,
    noRepliesRendered: 0,
    collapsed: false, // ignore for root
    comment, // ignore for root
    children,
  };
}

function pushComment(node, comment, pushToFront = false) {
  const newNode = new Node(comment, null, node);
  if (!node.children) {
    node.children = [];
  }
  if (pushToFront) {
    node.children.unshift(newNode);
  } else {
    node.children.push(newNode);
  }
  return newNode;
}

export function commentsTree(comments = [], root = new Node(null)) {
  if (comments === null || comments === undefined) {
    return root;
  }
  const partials = []; // array of roots
  for (const comment of comments) {
    const parentId = comment.parentId;
    if (parentId === null) {
      pushComment(root, comment);
    } else {
      let hit = searchTree(root, parentId, comment.depth);
      if (!hit) {
        for (const partial of partials) {
          hit = searchTree(partial, parentId, comment.depth);
          if (hit) {
            break;
          }
        }
      }
      if (hit) {
        pushComment(hit, comment);
      } else {
        partials.push(new Node(comment));
      }
    }
  }
  const merged = mergeTrees(root, partials);
  updateNoRendered(merged);
  return merged;
}

function updateNoRendered(root) {
  if (!root.children) {
    return 0;
  }
  let no = root.children.length;
  for (const child of root.children) {
    no += updateNoRendered(child);
  }
  root.noRepliesRendered = no;
  return no;
}

function mergeTrees(root, partials = []) {
  const all = [root, ...partials];
  for (let i = 0; i < partials.length; i++) {
    const { parentId, depth } = partials[i].comment;
    for (const item of all) {
      if (item === undefined) {
        continue;
      }
      const hit = searchTree(item, parentId, depth);
      if (hit) {
        if (!hit.children) {
          hit.children = [];
        }
        partials[i].parent = hit;
        hit.children.push({ ...partials[i] });
        delete partials[i];
        break;
      }
    }
  }
  // check
  for (const p of partials) {
    if (p !== undefined) {
      console.error("Comments tree: orphaned node");
    }
  }
  return root;
}

function updateNoReplies(node) {
  let parent = node.parent;
  while (parent) {
    if (parent.comment) {
      parent.comment.noReplies++; // could be root
    }
    parent.noRepliesRendered++;
    parent = parent.parent;
  }
  return node;
}

export function addComment(root, comment) {
  let node;
  if (comment.parentId === null) {
    node = pushComment(root, comment, true);
  } else {
    const hit = searchTree(root, comment.parentId, comment.depth);
    if (hit === null) {
      throw new Error(`Parent comment not found (commentId:${comment.id})`);
    }
    node = pushComment(hit, comment, true);
  }
  updateNoReplies(node);
  updateNoRendered(root);
  return node;
}

export function updateComment(root, comment) {
  const hit = searchTree(root, comment.id, comment.depth);
  if (hit === null) {
    throw new Error(`Comment not found (commentId:${comment.id})`);
  }
  hit.comment = {
    ...hit.comment,
    ...comment,
  };
  return hit;
}

export function countChildrenReplies(node) {
  let n = 0;
  if (node.children) {
    n = node.children.length;
    for (const child of node.children) {
      n += child.comment.noReplies;
    }
  }
  return n;
}
