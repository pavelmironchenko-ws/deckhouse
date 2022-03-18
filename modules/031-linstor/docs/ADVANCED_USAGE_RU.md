---
title: "Модуль linstor: примеры расширенной конфигурации"
---

Чтобы продолжить рекомендуется установить плагин [kubectl-linstor](https://github.com/piraeusdatastore/kubectl-linstor) или добавить bash-алиас:

```
alias linstor='kubectl exec -n d8-linstor deploy/linstor-controller -- linstor'
```

Дальнейшая настройка выполняется с помощью коммандной утилиты `linstor`

Настройка нод уже произведена автоматически. Для того чтобы начать использовать LINSTOR, вам нужно сделать две вещи:

- Создать пулы хранения
- Описать желаемые параметры в StorageClass

## Создание пулов хранения

отобразить список всех нод:
```
linstor node list
```

отобразить список всех доступных блочных устройств для хранения:
```
linstor physical-storage list
```

пример вывода:

```
+----------------------------------------------------------------+
| Size          | Rotational | Nodes                             |
|================================================================|
| 1920383410176 | False      | node01[/dev/nvme1n1,/dev/nvme0n1] |
| 1920383410176 | False      | node02[/dev/nvme1n1,/dev/nvme0n1] |
| 1920383410176 | False      | node03[/dev/nvme1n1,/dev/nvme0n1] |
+----------------------------------------------------------------+
```

> Внимание: здесь будут отображены только пустые девайсы без какой-либо разметки.
> Тем не менее, создание пулов хранения из партиций и других блочных девайсов также поддерживается.

> Вы также можете добавить уже существующий LVM пул в ваш кластер, для этого обратитесь к [FAQ](faq.html#как-добавить-существуюший-lvm-или-lvmthin-пул).


Создайте LVM или LVMThin пул из этих устройств:

- Чтобы создать LVM пул хранения из двух устройств на одной из нод можно следующим образом:
  
  ```
  linstor physical-storage create-device-pool lvm node01 /dev/nvme0n1 /dev/nvme1n1 --pool-name linstor_data --storage-pool lvm
  ```
  
- Чтобы создать ThinLVM пул хранения из двух устройств на одной из нод можно следующим образом:
  ```
  linstor physical-storage create-device-pool lvmthin node01 /dev/nvme0n1 /dev/nvme1n1 --pool-name data --storage-pool lvmthin
  ```

опции:
- `--pool-name` - имя VG/LV создаваемой на ноде
- `--storage-pool` - то как будет называться пул в LINSTOR

Вам необходимо создать несколько таких пулов для каждой вашей ноды, по возможности назовите их одинаково.

Как только пулы созданы, вы можете увидеть их выполнив:

```
linstor storage-pool list
```

пример вывода:

```
+---------------------------------------------------------------------------------------------------------------------------------+
| StoragePool          | Node   | Driver   | PoolName          | FreeCapacity | TotalCapacity | CanSnapshots | State | SharedName |
|=================================================================================================================================|
| DfltDisklessStorPool | node01 | DISKLESS |                   |              |               | False        | Ok    |            |
| DfltDisklessStorPool | node02 | DISKLESS |                   |              |               | False        | Ok    |            |
| DfltDisklessStorPool | node03 | DISKLESS |                   |              |               | False        | Ok    |            |
| lvmthin              | node01 | LVM_THIN | linstor_data/data |     3.49 TiB |      3.49 TiB | True         | Ok    |            |
| lvmthin              | node02 | LVM_THIN | linstor_data/data |     3.49 TiB |      3.49 TiB | True         | Ok    |            |
| lvmthin              | node03 | LVM_THIN | linstor_data/data |     3.49 TiB |      3.49 TiB | True         | Ok    |            |
+---------------------------------------------------------------------------------------------------------------------------------+
```

## Создание StorageClass

Теперь опишите желаемое количество реплик и имя пула в котором они будут создаваться в вашем StorageClass и примените его в Kubernetes:

```
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: linstor-r2
parameters:
  linstor.csi.linbit.com/storagePool: lvmthin
  linstor.csi.linbit.com/placementCount: "2"
allowVolumeExpansion: true
provisioner: linstor.csi.linbit.com
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
```

На этом конфигурацию можно считать законченной.