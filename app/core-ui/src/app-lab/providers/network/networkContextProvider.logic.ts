import {
  connectToWiFi,
  disconnectWiFi,
  getEthernetStatus,
  getInternetStatus,
  getNetworkList,
  getWiFiStatus,
} from '@cloud-editor-mono/domain/src/services/services-by-app/app-lab';
import {
  NetworkCredentials,
  NetworkItem,
  WiFiConnectionErrorCode,
} from '@cloud-editor-mono/ui-components/lib/components-by-app/app-lab';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { useShallow } from 'zustand/react/shallow';

import { BoardScopedQuery } from '../../boardScopedQuery';
import { useBoardLifecycleStore } from '../../store/boardLifecycle';
import { NetworkContextValue } from './networkContext';

export function useNetwork(): NetworkContextValue {
  const queryClient = useQueryClient();

  const [connectRequestErrorCode, setConnectRequestErrorCode] =
    useState<WiFiConnectionErrorCode>();

  const {
    mutate: connectToWifiNetwork,
    isLoading: connectRequestIsLoading,
    isSuccess: connectRequestIsSuccess,
    isError: connectRequestIsError,
    reset: resetConnectRequest,
  } = useMutation({
    mutationKey: ['connect-to-wifi-network'],
    mutationFn: async ({ name, password }: NetworkCredentials) => {
      await connectToWiFi(name, password);
    },
    onMutate: () => {
      setConnectRequestErrorCode(undefined);
      queryClient.setQueryData([BoardScopedQuery.WIFI_STATUS], 'connecting');
    },
    onSuccess: () => {
      setConnectRequestErrorCode(undefined);
      queryClient.setQueryData([BoardScopedQuery.WIFI_STATUS], 'connected');
    },
    onError: (error) => {
      const code = error instanceof Error ? error.message : undefined;
      setConnectRequestErrorCode(
        code === WiFiConnectionErrorCode.IncorrectPassword
          ? WiFiConnectionErrorCode.IncorrectPassword
          : WiFiConnectionErrorCode.ConnectionFailed,
      );
      queryClient.setQueryData([BoardScopedQuery.WIFI_STATUS], 'disconnected');
    },
    onSettled: () => {
      queryClient.invalidateQueries({
        queryKey: [BoardScopedQuery.WIFI_STATUS],
      });
    },
  });

  const { boardIsFlashing, boardIsReachable } = useBoardLifecycleStore(
    useShallow((state) => ({
      boardIsFlashing: state.boardIsFlashing,
      boardIsReachable: state.boardIsReachable,
    })),
  );

  const {
    mutateAsync: disconnectFromNetwork,
    isLoading: disconnectRequestIsLoading,
  } = useMutation({
    mutationFn: async () => {
      return disconnectWiFi();
    },
    onMutate: () => {
      resetConnectRequest();
      queryClient.setQueryData([BoardScopedQuery.WIFI_STATUS], 'disconnected');
    },
    onSettled: () => {
      queryClient.invalidateQueries({
        queryKey: [BoardScopedQuery.WIFI_STATUS],
      });
      queryClient.invalidateQueries({
        queryKey: [BoardScopedQuery.INTERNET_STATUS],
      });
    },
  });

  const {
    data: internetIsReachable,
    isLoading: isInternetStatusLoading,
    isSuccess: internetStatusChecked,
  } = useQuery(
    [BoardScopedQuery.INTERNET_STATUS],
    async () => getInternetStatus(),
    {
      retry: 3,
      refetchInterval: 3000,
      enabled: !boardIsFlashing && boardIsReachable,
    },
  );

  const [scanCount, setScanCount] = useState(0);
  const [scanningIsEnabled, setScanningIsEnabled] = useState(false);
  const shouldCheckManagedNetworkStatus =
    scanningIsEnabled &&
    internetStatusChecked &&
    internetIsReachable !== true;

  const {
    data: wiFiStatus,
    isLoading: isWiFiStatusLoading,
  } = useQuery([BoardScopedQuery.WIFI_STATUS], async () => getWiFiStatus(), {
    retry: 3,
    refetchInterval: 3000,
    enabled:
      !boardIsFlashing &&
      boardIsReachable &&
      shouldCheckManagedNetworkStatus &&
      !connectRequestIsLoading &&
      !disconnectRequestIsLoading,
  });

  const {
    data: ethernetStatus,
    isLoading: isEthernetStatusLoading,
  } = useQuery(
    [BoardScopedQuery.ETHERNET_STATUS],
    async () => getEthernetStatus(),
    {
      retry: 3,
      refetchInterval: 3000,
      enabled:
        !boardIsFlashing &&
        boardIsReachable &&
        shouldCheckManagedNetworkStatus &&
        !connectRequestIsLoading &&
        !disconnectRequestIsLoading,
    },
  );

  const isConnected = internetIsReachable === true;
  const {
    data: networkList,
    isFetching: isScanning,
    refetch: scanNetworkList,
  } = useQuery([BoardScopedQuery.NETWORK_LIST], getNetworkList, {
    onSuccess: (data) => {
      const list = data || [];
      setScanCount(list.length > 0 ? 8 : (c): number => c + 1);
    },
    // A Wi-Fi rescan can briefly drop an already-active connection, so never
    // trigger it automatically while online — only an explicit scanNetworkList()
    // call (e.g. the user asking to change networks) bypasses this and scans anyway.
    enabled: shouldCheckManagedNetworkStatus,
    refetchInterval:
      shouldCheckManagedNetworkStatus && scanCount < 8 ? 1500 : false,
  });

  const isStatusConnecting =
    wiFiStatus === 'connecting' || ethernetStatus === 'connecting';

  const [selectedNetwork, setSelectedNetwork] = useState<NetworkItem>();
  const [manualNetworkSetup, setManualNetworkSetup] = useState(false);

  useEffect(() => {
    resetConnectRequest();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedNetwork, manualNetworkSetup]);

  return {
    isScanning: isScanning || (shouldCheckManagedNetworkStatus && scanCount < 8),
    setScanningIsEnabled,
    networkList: networkList || [],
    isNetworkStatusLoading:
      isInternetStatusLoading ||
      (shouldCheckManagedNetworkStatus &&
        (isWiFiStatusLoading || isEthernetStatusLoading)),
    networkStatusChecked: internetStatusChecked,
    scanNetworkList,
    connectToWifiNetwork,
    disconnectFromNetwork,
    isConnected,
    isStatusConnecting,
    isConnecting:
      connectRequestIsLoading ||
      isStatusConnecting ||
      (connectRequestIsSuccess && !internetIsReachable),
    connectRequestIsSuccess,
    connectRequestIsError,
    connectRequestErrorCode,
    selectedNetwork,
    setSelectedNetwork,
    manualNetworkSetup,
    setManualNetworkSetup,
  };
}
