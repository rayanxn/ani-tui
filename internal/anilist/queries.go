package anilist

// searchAnimeQuery searches for anime with pagination
const searchAnimeQuery = `
query SearchAnime($search: String!, $page: Int) {
  Page(page: $page, perPage: 20) {
    pageInfo {
      total
      currentPage
      lastPage
      hasNextPage
    }
    media(search: $search, type: ANIME, sort: SEARCH_MATCH) {
      id
      title {
        romaji
        english
        native
      }
      format
      status
      episodes
      averageScore
      description(asHtml: false)
    }
  }
}
`

// getAnimeDetailsQuery retrieves full details for a specific anime
const getAnimeDetailsQuery = `
query GetAnimeDetails($id: Int!) {
  Media(id: $id, type: ANIME) {
    id
    title {
      romaji
      english
      native
    }
    description(asHtml: false)
    format
    status
    episodes
    duration
    averageScore
    genres
    studios(isMain: true) {
      nodes {
        name
      }
    }
    nextAiringEpisode {
      episode
      airingAt
      timeUntilAiring
    }
  }
}
`

// getUserListQuery retrieves a user's anime list
const getUserListQuery = `
query GetUserList($userId: Int!) {
  MediaListCollection(userId: $userId, type: ANIME) {
    lists {
      status
      entries {
        id
        status
        progress
        score
        media {
          id
          title {
            romaji
            english
            native
          }
          format
          status
          episodes
          averageScore
        }
      }
    }
  }
}
`

// updateProgressMutation updates the progress for an anime in the user's list
const updateProgressMutation = `
mutation UpdateProgress($mediaId: Int!, $progress: Int!, $status: MediaListStatus) {
  SaveMediaListEntry(mediaId: $mediaId, progress: $progress, status: $status) {
    id
    progress
    status
  }
}
`

// viewerQuery retrieves the authenticated user's information
const viewerQuery = `
query GetViewer {
  Viewer {
    id
    name
  }
}
`
