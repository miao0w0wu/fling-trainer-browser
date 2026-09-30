import { useEffect } from 'react'
import { Events } from '@wailsio/runtime'
import { useAppStore } from '../store/appStore'

export function useDownloadProgress() {
  const setDownloadProgress = useAppStore((state) => state.setDownloadProgress)

  useEffect(() => {
    const off = Events.On('download:progress', (event) => {
      setDownloadProgress(event.data)
    })
    return () => off()
  }, [setDownloadProgress])
}
